package main

func create(args []string) {
	opts := parseArgs(args)

	// Preamble, depending on the format
	if opts.Format == "" || opts.Format == "yaml" {
		log.Println(`apiVersion: v1
kind: Secret
metadata:
name: mysecret
type: Opaque
data:`)
	}

	// Give all handlers a go to write to the secret data
	data := make(map[string][]byte, 0)
	for _, h := range opts.Handlers {
		h.Populate(data)
	}

	// Print all values in the correct format
	for key, value := range data {
		switch opts.Format {
		case "env":
			log.Printf("%s=%s", key, string(value))
		default:
			log.Printf("  %s: %s", key, string(value))
		}
	}
}
