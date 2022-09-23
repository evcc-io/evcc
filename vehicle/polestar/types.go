package polestar

type Consumer struct {
	// Type                  string `graphql:"__typename"`
	MyStarConsumerDetails `graphql:"... on MyStarConsumer"`
}

type MyStarConsumerDetails struct {
	// Type      string `graphql:"__typename"`
	Email     string
	FirstName string
	LastName  string
}

type ConsumerCar struct {
	// Type             string `graphql:"__typename"`
	MyStarCarDetails `graphql:"... on MyStarCar"`
}

type MyStarCarDetails struct {
	Id int
	// ConsumerId       string
	// Engine string
	// Exterior         string
	// ExteriorCode     string
	// ExteriorImageUrl string
	// Gearbox          string
	// Interior         string
	// InteriorCode     string
	// InteriorImageUrl string
	Model     string
	ModelYear int
	// Package          string
	// PackageCode      string
	// PdfUrl           string
	// Status string
	// Steering         string
	Vin string
	// Wheels           string
	// WheelsCode       string
}
