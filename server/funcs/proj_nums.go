package funcs

import (
	"errors"
	"server/database"
	"server/models"
	"sort"

	"go.mongodb.org/mongo-driver/mongo"
)

// ReassignNumsInOrder assigns project numbers in order.
func ReassignNumsInOrder(db *mongo.Database) error {
	err := database.WithTransaction(db, func(sc mongo.SessionContext) error {
		// Get all the projects from the database
		projects, err := database.FindAllProjects(db, sc)
		if err != nil {
			return errors.New("error getting projects from database: " + err.Error())
		}

		// If projects is empty, send OK
		if len(projects) == 0 {
			return nil
		}

		// Sort projets by table num
		sort.Sort(models.ByTableNumber(projects))

		// Get the options from the database
		options, err := database.GetOptions(db, sc)
		if err != nil {
			return errors.New("error getting options from database: " + err.Error())
		}

		// Set init table num to 0
		options.CurrTableNum = 0

		// Loop through all projects
		for _, project := range projects {
			project.Location = options.GetNextIncrTableNum()
		}

		// Update the options in the database
		err = database.UpdateCurrTableNum(db, sc, options)
		if err != nil {
			return errors.New("error updating options in database: " + err.Error())
		}

		// Update all projects in the database
		err = database.UpdateProjects(db, projects)
		if err != nil {
			return errors.New("error updating projects in database: " + err.Error())
		}
		return nil
	})

	return err
}

func ReassignNumsByGroup(db *mongo.Database) error {
	err := database.WithTransaction(db, func(sc mongo.SessionContext) error {
		// Get all the projects from the database
		projects, err := database.FindAllProjects(db, sc)
		if err != nil {
			return errors.New("error getting projects from database: " + err.Error())
		}

		// Get the options from the database
		options, err := database.GetOptions(db, sc)
		if err != nil {
			return errors.New("error getting options from database: " + err.Error())
		}

		// Sort projets by table num
		sort.Sort(models.ByTableNumber(projects))

		// Set init table num to 0
		options.CurrTableNum = 0

		// Create group table numbers slice
		options.GroupTableNums = make([]int64, options.NumGroups)

		// Fill group table numbers slice with numbers corresponding to the group
		for i := range options.GroupTableNums {
			if i == 0 {
				options.GroupTableNums[0] = 0
				continue
			}
			options.GroupTableNums[i] = options.GroupSizes[i-1] + options.GroupTableNums[i-1]
		}

		// Loop through all projects
		for _, project := range projects {
			project.Group, project.Location = options.GetNextGroupTableNum()
		}

		// Update the options in the database
		err = database.UpdateCurrTableNum(db, sc, options)
		if err != nil {
			return errors.New("error updating options in database: " + err.Error())
		}

		// Don't update if there are no projects
		if len(projects) == 0 {
			return nil
		}

		// Update all projects in the database
		err = database.UpdateProjects(db, projects)
		if err != nil {
			return errors.New("error updating projects in database: " + err.Error())
		}
		return nil
	})

	return err
}

// GetNextTableNum gets the group and table number for the next project added.
// If groups is not enabled, it will only return a table number (first return value will be null).
func GetNextTableNum(db *mongo.Database, op *models.Options) (int64, int64) {
	if op.MultiGroup {
		group, table := op.GetNextGroupTableNum()
		return group, table
	} else {
		table := op.GetNextIncrTableNum()
		return 0, table
	}
}

// IncrementJudgeGroupNum increments every single judges' group number.
func IncrementJudgeGroupNum(db *mongo.Database) error {
	err := database.WithTransaction(db, func(sc mongo.SessionContext) error {
		// Get the options from the database
		options, err := database.GetOptions(db, sc)
		if err != nil {
			return errors.New("error getting options from database: " + err.Error())
		}

		// Get the judges from the database
		judges, err := database.FindAllJudges(db, sc)
		if err != nil {
			return errors.New("error getting judges from database: " + err.Error())
		}

		// Increment the group number for each judge
		for _, judge := range judges {
			judge.Group = (judge.Group + 1) % options.NumGroups
		}

		// Update all judges in the database
		// TODO: Can we write a function that will only update that field instead of have to pass ALL the judge data to the db :(
		err = database.UpdateJudgesWithTx(db, sc, judges)
		if err != nil {
			return errors.New("error updating judges in database: " + err.Error())
		}

		// Increment the manual switch count in the database
		err = database.IncrementManualSwitches(db, sc)
		if err != nil {
			return errors.New("error incrementing manual switches in database: " + err.Error())
		}

		return nil
	})

	return err
}
