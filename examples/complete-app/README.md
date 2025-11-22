# E2E Test Guide

This directory contains a complete example to test the operator capabilities.

## Prerequisites

1. Deploy the operator
2. Apply the shared resources (simulates your infrastructure):
   ```bash
   kubectl apply -f 00-shared-resources.yaml
   ```

## How to Test

1. **Push the base app to a Git repository**
   
   The `deployment.yaml` in `base/` expects certain secrets and configmaps to exist. 
   You need to commit this directory to a Git repository that ArgoCD can access.

2. **Update the EphemeralApplication**
   
   Edit `ephemeral-app.yaml` and update `repoURL` to point to your repository where you pushed `base/`.

3. **Create the Ephemeral Environment**

   ```bash
   kubectl apply -f ephemeral-app.yaml
   ```

4. **Verify Injection**

   Wait for the application to be healthy:
   ```bash
   kubectl get ephapp e2e-demo
   ```

   Get the logs to verify environment variables were injected correctly:
   ```bash
   # Get the namespace name
   NS=$(kubectl get ephapp e2e-demo -o jsonpath='{.status.namespace}')
   
   # Check logs
   kubectl logs -n $NS -l app=demo-app
   ```

   You should see:
   ```
   Starting app...
   DB_USERNAME=admin
   DB_PASSWORD=super-secret-password
   APP_REGION=eu-west-1
   APP_MODE=ephemeral-test
   ```

## What just happened?

1. Operator created a new namespace `ephemeral-xxxx`
2. Operator copied `database-creds` secret from `shared-infrastructure` namespace
3. Operator copied `global-config` configmap from `shared-infrastructure` namespace
4. Operator created `app-settings` configmap with inline values
5. ArgoCD deployed the app from Git
6. The app successfully started because all referenced secrets/configmaps were present!

