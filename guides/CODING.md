# Coding standards

## Running code

Although go has some command (go run, go build, etc.), we are using makefile recipes instead.

We run code from the top level directory.

## Logging policy

Errors are logged at the first function checking for errors in code not written in this repository.

So:

```go
func example() (string, err){
  x, err := not_my_lib()
  if err != nil {
    log.Printf("Error in not_my_lib: %v", err)
    return "", err
  }
  return x, nil
}
```

But:

```
func example2() (string, err) {
  x, err := example()
  if err != nil {
    // would already have been logged in example()
    return "", err
  }
  return x, nil
}

```


## Notes on concurrency

Initially, the code was running gdelt and google news together, using goroutines. However, this resulted in some weirdness. Instead now we have separate processes for each source.
