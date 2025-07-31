package migration_0001

import "fmt"

func main() {
	success, err := Migrate()
	if err != nil {
		fmt.Println("Migration failed:", err)
		return
	}
	if success {
		fmt.Println("Migration successful")
	}

}
