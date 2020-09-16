# Hacking

## Operator Lifecycle Manager

### Prerequisites

1. Install OLM and Operator Marketplace:
   ```
   $ ./deploy/olm/install.sh
   ```
2. Install Operator Courier:
   ```
   $ pip3 install operator-courier
   ```
3. [Sign up](https://quay.io) for a free Quay.io account if you're a new user.

### Building the OLM Bundle

The Marketplace Operator can import Operators from external data stores.
To test Starboard Operator deployment we'll be using Quay.io to host the OLM bundle.

1. Perform linting
   ```
   $ operator-courier verify deploy/olm/bundle
   ```
2. Retrieve a Quay.io token
   ```
   QUAY_USERNAME=<your quay.io username>
   QUAY_PASSWORD=<your quay.io password>
   QUAY_URL=https://quay.io/cnr/api/v1/users/login

   QUAY_TOKEN=$(curl -s -H "Content-Type: application/json" -XPOST $QUAY_URL -d \
     '{"user":{"username":"'"${QUAY_USERNAME}"'","password": "'"${QUAY_PASSWORD}"'"}}' |
     jq -r .token)
   ```
3. Push the bundle to Quay.io
   ```
   BUNDLE_SRC_DIR=deploy/olm
   QUAY_NAMESPACE=<quay.io namespace>
   PACKAGE_NAME=starboard-operator
   PACKAGE_VERSION=0.0.1

   $ operator-courier push "$BUNDLE_SRC_DIR" "$QUAY_NAMESPACE" \
     "$PACKAGE_NAME" "$PACKAGE_VERSION" "$QUAY_TOKEN"
   ```

### Running Locally

1. Create the OperatorSource

   An OperatorSource resource defines the external data store used to host Operator bundles. In this case, you will be
   defining an OperatorSource to point to your Quay.io account, which will provide access to its hosted OLM bundles.

   ```
   QUAY_FULL_NAME=<your quay.io full name>
   ```

   ```
   $ cat << EOF | kubectl apply -f -
   apiVersion: operators.coreos.com/v1
   kind: OperatorSource
   metadata:
     name: $QUAY_USERNAME-operators
     namespace: marketplace
   spec:
     type: appregistry
     endpoint: https://quay.io/cnr
     displayName: "$QUAY_FULL_NAME Quay.io Applications"
     publisher: "$QUAY_FULL_NAME"
     registryNamespace: "$QUAY_USERNAME"
   EOF
   ```

   Verify that the OperatorSource was deployed correctly

   ```
   $ kubectl get operatorsources -n marketplace
   NAME                           TYPE          ENDPOINT              REGISTRY      STATUS      MESSAGE                                       AGE
   danielpacak-operators          appregistry   https://quay.io/cnr   danielpacak   Succeeded   The object has been successfully reconciled   24h
   ```

   Additionally, the OperatorSource creation results in the creation of a CatalogSource:
   
   ```
   kubectl get catalogsources -n marketplace
   NAME                    TYPE   PUBLISHER      AGE
   danielpacak-operators   grpc   Daniel Pacak   14m
   ```
2. Create the OperatorGroup

   You'll need an OperatorGroup to denote which namespaces the Operator should watch. It must exist in the namespace
   where you want to deploy the Operator.
   
   ```
   $ cat << EOF | kubectl apply -f -
   apiVersion: operators.coreos.com/v1alpha2
   kind: OperatorGroup
   metadata:
     name: workloads-og
     namespace: marketplace
   spec:
     targetNamespaces:
     - workloads
   EOF
   ```
3. Create the Subscription

   A subscription links the previous steps together by selecting an Operator and one of its channels. OLM uses this
   information to start the corresponding Operator pod. The following example creates a new subscription to the `alpha`
   channel for the Starboard Operator:
   
   ```
   cat << EOF | kubectl apply -f -
   apiVersion: operators.coreos.com/v1alpha1
   kind: Subscription
   metadata:
     name: starboard-operator
     namespace: marketplace
   spec:
     channel: alpha
     name: starboard-operator
     source: $QUAY_NAMESPACE-operators
     sourceNamespace: marketplace
   EOF
   ```

   OLM will be notified of the new subscription and will start the Operator pod in the marketplace namespace:

   ```
   $ kubectl get pod -n marketplace
   NAME                                            READY   STATUS    RESTARTS   AGE
   danielpacak-operators-6df86ff9c6-zkmqg          1/1     Running   0          11m
   marketplace-operator-698ddc5c67-zqqcg           1/1     Running   0          14m
   starboard-operator-c459584f7-9wwgt              1/1     Running   0          2m10s
   upstream-community-operators-6f546f47bb-pwr9q   1/1     Running   0          13m
   ```
