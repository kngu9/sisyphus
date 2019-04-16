// Copyright 2019 CanonicalLtd

package simulation

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/zaputil"
	"github.com/juju/zaputil/zapctx"
	"gopkg.in/macaroon-bakery.v1/httpbakery"

	"github.com/cloud-green/sisyphus/config"
	"github.com/cloud-green/sisyphus/simulation/call"
)

var (
	r *rand.Rand

	contextDoneError = errors.New("done")
)

// CallBackend defines the interface used by the simulation to perform
// state transition calls.
type CallBackend interface {
	Do(context.Context, config.Call, call.Attributes) (call.Attributes, error)
}

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// New returns a new simulation based on the provided configuration.
func New(config config.Config, callBackend CallBackend) (*Simulation, error) {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Simulation{
		Config:      config,
		Attributes:  config.Constants,
		CallBackend: callBackend,
		ctx:         ctx,
		stop:        cancel,
		client:      httpbakery.NewClient(),
	}

	// start creating root entity sets
	for _, entitySet := range config.RootEntities {
		newEntitySet(ctx, entitySet, s.Attributes, s)
	}

	s.wg.Wait()

	return s, nil
}

// Simulation represents a simulation
type Simulation struct {
	config.Config
	CallBackend
	call.Attributes

	ctx    context.Context
	wg     sync.WaitGroup
	errors chan error
	stop   func()
	client *httpbakery.Client
}

// add means that a go routing should be added to the wait group
func (s *Simulation) add() {
	s.wg.Add(1)
}

// done means that a go routing has exited
func (s *Simulation) done() {
	s.wg.Done()
}

// error means an error occured during execution of one go routine
// and the undelying context should be cancelled.
func (s *Simulation) error(err error) {
	if errors.Cause(err) != contextDoneError {
		zapctx.Error(s.ctx, "an error occured", zaputil.Error(err))
	}
	s.stop()
}

func newEntitySet(ctx context.Context, config config.EntitySet, attributes call.Attributes, sim *Simulation) {
	es := &entitySet{
		EntitySet:  config,
		attributes: attributes,
	}
	sim.add()
	go func(ctx context.Context) {
		defer sim.done()
		es.create(ctx, sim)
	}(ctx)
}

type entitySet struct {
	config.EntitySet
	attributes call.Attributes
}

func (e *entitySet) create(ctx context.Context, sim *Simulation) {
	// check that the configuration for the named entity exists
	cfg, ok := sim.Entities[e.Entity]
	if !ok {
		sim.error(errors.NotFoundf("entity %q", e.Entity))
		return
	}
	// determine how many entities are to be created
	c := cardinality(e.Cardinality)
	numberOfEntities, err := c.Value(e.attributes)
	if err != nil {
		sim.error(errors.Trace(err))
		return
	}
	// create the timer that determines the cadence of entity
	// creation
	timer := newTimer(e.Timer)
	for i := 0; i < numberOfEntities; i++ {
		err = timer.Next(ctx)
		if err != nil {
			sim.error(errors.Trace(err))
			return
		}
		createEntity(ctx, cfg, e.attributes, sim)
	}
	return
}

func createEntity(ctx context.Context, config config.Entity, attributes call.Attributes, sim *Simulation) {
	// first we copy the attribute map
	attrs := call.Attributes(make(map[string]interface{}, len(attributes)))
	for key, value := range attributes {
		attrs[key] = value
	}
	// the we sample the entities attributes
	for key, config := range config.Attributes {
		distribution := &AttributeDistribution{
			Attribute: config,
		}
		value, err := distribution.Sample()
		if err != nil {
			sim.error(errors.Trace(err))
			return
		}
		// and add sampled values to the set of attributes (as strings)
		attrs[key] = fmt.Sprintf("%f", value)
	}

	// if there are any subordinate entities, we create them
	for _, esConfig := range config.Subordinates {
		es := &entitySet{
			EntitySet:  esConfig,
			attributes: attrs,
		}
		sim.add()
		go func() {
			defer sim.done()
			es.create(ctx, sim)
		}()
	}

	// if an initial state is defined, we create it and run the state simulation
	if config.InitialState != "" {
		stateConfig, ok := sim.States[config.InitialState]
		if !ok {
			sim.error(errors.NotFoundf("state %q", config.InitialState))
			return
		}
		s := &State{
			State:      stateConfig,
			Attributes: attrs,
		}
		sim.add()
		go func() {
			defer sim.done()
			s.run(ctx, sim)
		}()
	}

}

type State struct {
	config.State
	call.Attributes
}

func (s *State) run(ctx context.Context, sim *Simulation) {
	// if there are no specified transtions, we just
	// return and end the simulation
	if len(s.Transitions) == 0 {
		return
	}

	// create a time that defines the transition cadence
	timer := newTimer(s.Timer)
	// wait for the timer to fire
	err := timer.Next(ctx)
	if err != nil {
		sim.error(errors.Trace(err))
		return
	}
	// calculate the sum of transition weigths
	sum := 0.0
	for _, transition := range s.Transitions {
		if transition.Probability < 0 {
			sim.error(errors.Errorf("negative transition probability %v", transition.Probability))
			return
		}
		sum += transition.Probability
	}
	if sum == 0 {
		sim.error(errors.Errorf("sum of transition probabilities is 0"))
		return
	}
	// create a random number [0 .. sum]
	randomNumber := sum * r.Float64()
	for _, transition := range s.Transitions {
		// subtract the transition weigth
		randomNumber -= transition.Probability
		// if we reached 0 (or less) we choose this transition
		if randomNumber <= 0 {
			nextStateName := transition.State

			attributes, err := sim.Do(ctx, transition.Call, s.Attributes)
			if err != nil {
				attributes["error"] = errors.Details(err)
				nextStateName = transition.OnFailure
			}
			nextStateConfig, ok := sim.States[nextStateName]
			if !ok {
				sim.error(errors.NotFoundf("state %q", nextStateName))
				return
			}

			nextState := &State{
				State:      nextStateConfig,
				Attributes: attributes,
			}
			go func() {
				sim.add()
				defer sim.done()
				nextState.run(ctx, sim)
			}()
		}
	}
}

type AttributeDistribution struct {
	config.Attribute
}

// Sample return a sample float64 from the attribute distribution
func (a *AttributeDistribution) Sample() (interface{}, error) {
	switch a.Type {
	case config.ConstantAttributeType:
		return a.Value, nil
	case config.RandomAttributeType:
		return a.Min + (a.Max-a.Min)*r.Float64(), nil
	case config.PowerAttributeType:
		v := r.Float64()
		return math.Pow((math.Pow(a.Max, a.N+1)-math.Pow(a.Min, a.N+1))*v+math.Pow(a.Min, a.N+1), (1 / (a.N + 1))), nil
	case config.ConstantStringAttributeType:
		return a.StringValue, nil
	case config.RandomStringAttributeType:
		if a.StringValue == "" {
			uuid, err := utils.NewUUID()
			if err != nil {
				return nil, errors.Trace(err)
			}
			return uuid.String(), nil
		}
		return fmt.Sprintf("%s%d", a.StringValue, int(math.Floor(a.Min+(a.Max-a.Min)*r.Float64()))), nil
	}

	return 0, nil
}

func newTimer(c config.Timer) *timer {
	return &timer{
		Timer: c,
		C:     make(chan time.Time, 1),
	}
}

type timer struct {
	config.Timer

	C chan time.Time
}

func (t *timer) Next(ctx context.Context) error {
	var duration time.Duration
	switch t.Type {
	case config.FixedTimer:
		duration = t.Interval
	case config.RandomTimer:
		duration = time.Duration(int64(t.Min) + rand.Int63n(int64(t.Max-t.Min)))
	case "":
		duration = 0
	}
	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return contextDoneError
	}
}

type cardinality string

// Value returns the value of the cardinality. It may be a constant integer
// or name of an attribute.
func (c *cardinality) Value(attributes call.Attributes) (int, error) {
	value, ok := attributes[string(*c)]
	if !ok {
		if *c == "" {
			return 1, nil
		}
		return strconv.Atoi(string(*c))
	}
	switch value.(type) {
	case int:
		return value.(int), nil
	case string:
		return strconv.Atoi(value.(string))
	default:
		return -1, errors.Errorf("unknown type: expected int or string")
	}
}
