package main

func main() {
	avFlags := NewFFsFlags()
	exists := avFlags.HasFile()
	if exists {
		avFlags.Load()
	}

	avFlags.Parse()

	avFlags.Print()

	info()
}
