# reset kubernetes installation

## remove all dory service when install failure

### stop and remove {{ $.imageRepo.type }} services

```shell script
helm -n {{ $.imageRepo.namespace }} uninstall {{ $.imageRepo.namespace }}
```

### stop and remove dory services

```shell script
# cd to readme directory
kubectl delete -f {{ $.dory.namespace }}
kubectl delete -f {{ $.imageRepo.namespace }}
kubectl delete -f .
```

## about dory services data

- dory services data located at: `{{ $.rootDir }}`