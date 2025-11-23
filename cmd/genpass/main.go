package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		panic(err)
	}
	fmt.Println("Password:", password)
	fmt.Println("Hash:", string(hash))
	fmt.Println("\nRun this SQL in Supabase:")
	fmt.Printf("UPDATE users SET password_hash = '%s' WHERE email IN ('godfather@test.com', 'doctor@test.com', 'patient@test.com', 'junior@test.com', 'admin@test.com');\n", string(hash))
}
