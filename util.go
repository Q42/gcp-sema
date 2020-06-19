package main

// prints stack
func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
