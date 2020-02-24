# swfdemo

### go install

```
go install -ldflags "-X 'github.com/qingsong-he/ce.DefaultVersion=`git rev-parse --short HEAD`' -X 'github.com/qingsong-he/ce.DefaultFrom=swfdemo'" github.com/qingsong-he/swfdemo
```