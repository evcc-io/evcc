# APIs

`/config/types/<class>`
`/config/templates/<class>`

## `/config/site/`

Structure:

```json
{
	"title": "String",
	"meters": {
		"grid", "Meter",
		"pv", "Meter",
		"battery", "Meter"
	}
}
```

## `/config/loadpoint/<id>/`

Structure:

```json
{
	"title": "String",
	"meters": {
		"charge", "Meter"
	}
}
```
