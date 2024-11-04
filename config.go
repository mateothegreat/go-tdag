package main

type Test struct {
	Name    string `yaml:"name" json:"name"`
	Command string `yaml:"command" json:"command"`
}

type Scenario struct {
	Name  string `yaml:"name" json:"name"`
	Tests []Test `yaml:"tests" json:"tests"`
}

type Config struct {
	Scenarios []Scenario `yaml:"scenarios" json:"scenarios"`
}
