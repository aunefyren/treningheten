package utilities

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

func PrintASCII() {
	fmt.Println(``)
	fmt.Println(`T R E N I N G H E T E N`)
	fmt.Println(``)
	return
}

func ValidatePasswordFormat(password string) (bool, string, error) {
	requirements := "Password must have a minimum of eight characters, at least one uppercase letter, one lowercase letter and one number."

	if len(password) < 8 {
		return false, requirements, nil
	}

	match, err := regexp.Match(`[A-ZÆØÅ]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	match, err = regexp.Match(`[a-zæøå]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	match, err = regexp.Match(`[0-9]{1,20}`, []byte(password))
	if err != nil {
		return false, requirements, err
	} else if !match {
		return false, requirements, nil
	}

	return true, requirements, nil
}

func FindNextSunday(poinInTime time.Time) (time.Time, error) {

	sundayDate := time.Time{}

	// Find sunday
	if poinInTime.Weekday() == 0 {
		sundayDate = poinInTime
	} else {
		nextDate := poinInTime

		for i := 0; i < 8; i++ {
			nextDate = nextDate.AddDate(0, 0, +1)
			if nextDate.Weekday() == 0 {
				sundayDate = nextDate
				break
			}
		}

	}

	if sundayDate.Weekday() == 0 {
		return sundayDate, nil
	}

	return time.Time{}, errors.New("Failed to find next sunday for date.")
}

func FindEarlierMonday(poinInTime time.Time) (time.Time, error) {

	mondayDate := time.Time{}

	// Find monday
	if poinInTime.Weekday() == 1 {
		mondayDate = poinInTime
	} else {
		previousDate := poinInTime

		for i := 0; i < 8; i++ {
			previousDate = previousDate.AddDate(0, 0, -1)
			if previousDate.Weekday() == 1 {
				mondayDate = previousDate
				break
			}
		}

	}

	if mondayDate.Weekday() == 1 {
		return mondayDate, nil
	}

	return time.Time{}, errors.New("Failed to find earlier monday for date.")
}

func RemoveIntFromArray(originalArray []int, intToRemove int) []int {

	newArray := []int{}

	for _, intNumber := range originalArray {
		if intNumber != intToRemove {
			newArray = append(newArray, intNumber)
		}
	}

	return newArray

}
