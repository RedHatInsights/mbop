#!/bin/bash

set -e

IMAGE="quay.io/cloudservices/mbop"  # the image location on quay

# We will run rbac smoke tests to validate mbop is working

APP_NAME="rbac"  # name of app-sre "application" folder this component lives in
COMPONENT_NAME="rbac"  # name of app-sre "resourceTemplate" in deploy.yaml for this component
IQE_PLUGINS="rbac,rbac_frontend"  # name of the IQE plugin for this APP
IQE_FILTER_EXPRESSION="test_permissions_sanity or test_mua_navigate"  # expression passed to pytest '-k'
IQE_CJI_TIMEOUT="10m"  # This is the time to wait for smoke test to complete or fail
DEPLOY_FRONTENDS="true"
IQE_SELENIUM="true"
IQE_ENV="ephemeral"
REF_ENV="insights-stage"
NAMESPACE_POOL="default"

# Install bonfire repo/initialize
CICD_URL=https://raw.githubusercontent.com/RedHatInsights/cicd-tools/main/bootstrap.sh
curl -s "$CICD_URL" > .cicd_bootstrap.sh && source .cicd_bootstrap.sh

# Build pr check quay image
source $CICD_ROOT/build.sh

# Do not 'source $CICD_ROOT/deploy_ephemeral_env.sh' here because this is a slightly different
# test scenario where we cannot override the mbop image tag at deployment time, run our own
# deploy command ...
source ${CICD_ROOT}/_common_deploy_logic.sh

set -x

export BONFIRE_NS_REQUESTER="${JOB_NAME}-${BUILD_NUMBER}"
export NAMESPACE=$(bonfire namespace reserve --pool ${NAMESPACE_POOL})
SMOKE_NAMESPACE=$NAMESPACE  # track which namespace was used here for 'teardown' in common_deploy_logic

bonfire deploy \
    ${APP_NAME} \
    --source=appsre \
    --ref-env ${REF_ENV} \
    --namespace ${NAMESPACE} \
    --timeout ${DEPLOY_TIMEOUT} \
    --frontends ${DEPLOY_FRONTENDS} \
    ${COMPONENTS_ARG} \
    ${COMPONENTS_RESOURCES_ARG}

set +x

# Update mbop in this environment to use the newly built PR image
CLOWDENV_NAME="env-$NAMESPACE"
kubectl patch clowdenvironment ${CLOWDENV_NAME} --type='merge' -p '{"spec":{"providers":{"web":{"images":{"mockBop":"'${IMAGE}:${IMAGE_TAG}'"}}}}}'
kubectl rollout status deployment/${CLOWDENV_NAME}-mbop -n $NAMESPACE

# Run rbac test
source $CICD_ROOT/cji_smoke_test.sh
