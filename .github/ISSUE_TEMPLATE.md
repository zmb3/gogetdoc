Before submitting an issue, please make sure:
- [ ] You're running Go 1.6 or later
- [ ] You've tried installing with `go get -u` to update dependencies

If you see the following error, you need to update to Go 1.6+:
```
$ go get github.com/zmb3/gogetdoc
# github.com/zmb3/gogetdoc
./ident.go:142: c.Val().ExactString undefined (type constant.Value has no field or method ExactString)
```
