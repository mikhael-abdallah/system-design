package main

import "github.com/google/uuid"

type User struct {
	ID   uuid.UUID `json:"id" bson:"_id"`
	Name string    `json:"name" bson:"name"`
	Data string    `json:"data" bson:"data"`
}