## API encoding

|               | socket               | /api/status                   | mqtt      | push          | influx        | rest |
| ------------- | -------------------- | ----------------------------- | --------- | ------------- | ------------- | ---- |
| source        | chan<br>encode.New() | cache.State()<br>encode.New() | chan      | cache.All()   | chan          | -    |
| time.Time     | rfc3999/nil          | rfc3999/nil                   | rfc3999   | time.Time     | time.Time     | ?    |
| time.Duration | s                    | s                             | s         | time.Duration | time.Duration | s    |
| fmt.Stringer  | string               | string                        | string    | string        | string        | ?    |
| struct{}      | ?                    | recursive                     | recursive | n/a           | ?             | ?    |
