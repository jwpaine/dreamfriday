package models

import (
	database "dreamfriday/database"
	"fmt"
	"log"
)

// Save stores or updates the user in the "Users" bucket.
func (u *User) Save() error {
	return database.Put("Users", u.Address, u)
}

// GetUser retrieves a user by their address.
func GetUser(address string) (*User, error) {
	var user User
	err := database.Get("Users", address, &user)
	if err != nil {
		log.Println("Failed to get user:", err)
		return nil, fmt.Errorf("user not found: %w", err)
	}
	log.Println("Fetched user data for handle:", address)
	return &user, nil
}

// DeleteUser removes a user from the database.
func DeleteUser(address string) error {
	return database.Delete("Users", address)
}

// AddSiteToUser adds a new site name to the user's collection of sites.
func AddSiteToUser(userAddress, siteName string) error {
	user, err := GetUser(userAddress)
	if err != nil {
		// If user doesn't exist, create a new one with the given site
		user = &User{
			Address: userAddress,
			Sites:   []string{siteName},
		}
	} else {
		// Avoid duplicate site names
		for _, s := range user.Sites {
			if s == siteName {
				return fmt.Errorf("site %q already exists for user %s", siteName, userAddress)
			}
		}
		// Append the new site name
		user.Sites = append(user.Sites, siteName)
	}

	// Save the updated user record
	return user.Save()
}
