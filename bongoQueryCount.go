package main

import (
	"github.com/go-bongo/bongo"
)

// CountResults - Add a function to the bongo ResultSet to count the amount of results
func CountResults(r *bongo.ResultSet) (int, error) {

	// Get count on a different session to avoid blocking
	sess := r.Collection.Connection.Session.Copy()

	count, err := sess.DB(r.Collection.Database).C(r.Collection.Name).Find(r.Params).Count()
	sess.Close()

	if err != nil {
		return 0, err
	}

	return count, nil
}
