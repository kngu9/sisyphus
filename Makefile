SISYPHUS_VERSION := $(shell awk '/version:/ {print $$2}' snap/snapcraft.yaml | head -1 | sed "s/'//g")

.PHONY: all
all: snap lint charm

.PHONY: lint
lint:
	flake8

.PHONY: snap
snap: sisyphus_$(KAFKA_VERSION)_amd64.snap

.PHONY: charm
charm: charm/builds/sisyphus

charm/builds/sisyphus:
	$(MAKE) -C charm/sisyphus

.PHONY: clean-charm
clean-charm:
	$(RM) -r charm/builds charm/deps
