
# dev: clean-kind ## Run local dev with Skaffold, watching for code changes. Deletes and recreates the test cluster.
# > kind create cluster --config=k8s/dev/skaffold/kind.yaml --name=ara-local-dev
# > skaffold debug -f skaffold.debug.yaml -p $(SKAFFOLD_PROFILE)
# .PHONY: dev

# Default Skaffold profile
SKAFFOLD_PROFILE ?= default
# HOME = $(HOME)


clean-kind: ## Deletes the local dev cluster created by Kind.
	kind delete cluster --name=lease-cluster
.PHONY: clean-kind

KUSTOMIZE ?= /usr/local/bin/kustomize

kustomize: ## Ensure Kustomize is installed
	@which $(KUSTOMIZE) >/dev/null || (echo "Kustomize is not installed at $(KUSTOMIZE). Please install it." && exit 1)


merge: kustomize
	$(KUSTOMIZE) build crd |go run ./crd/template/pre_helm.go |go run ./merge/merge.go > all-merged.yaml	

dev: clean-kind ## Run local dev with Skaffold, watching for code changes. Deletes and recreates the test cluster.
	kind create cluster --name=lease-cluster --config=cluster-config.yaml
	skaffold dev -p $(SKAFFOLD_PROFILE)
.PHONY: dev helm
