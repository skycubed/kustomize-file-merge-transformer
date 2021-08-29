PLUGIN_DIR=plugin/skycubed.github.com/v1/filemergetransformer
PLUGIN_OUT=$(XDG_CONFIG_HOME)/kustomize/$(PLUGIN_DIR)/FileMergeTransformer

$(PLUGIN_OUT): cmd/FileMergeTransformer/main.go
	mkdir -p $(dir $(PLUGIN_OUT))
	go build -o $(PLUGIN_OUT) $<

.PHONY: build
build: $(PLUGIN_OUT)
	@true

.PHONY: install_kustomize
install_kustomize:
	cd /tmp && GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3

.PHONY: example-base
example-base:
	@kustomize build example/base

.PHONY: example-transform
example-transform: build
	@kustomize build --enable_alpha_plugins example/overlays/development

.PHONY: deploy-example
deploy-example: build
	@kustomize build --enable_alpha_plugins example/overlays/development | kubectl apply -f -
