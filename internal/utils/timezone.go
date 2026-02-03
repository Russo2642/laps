package utils

import "time"

var (
	// AlmatyLocation - часовой пояс Алматы (UTC+5)
	AlmatyLocation *time.Location
)

func init() {
	var err error
	AlmatyLocation, err = time.LoadLocation("Asia/Almaty")
	if err != nil {
		AlmatyLocation = time.FixedZone("Asia/Almaty", 5*60*60)
	}
}

func NowInAlmaty() time.Time {
	return time.Now().In(AlmatyLocation)
}

func IsTodayInAlmaty(date time.Time) bool {
	now := NowInAlmaty()
	dateInAlmaty := date.In(AlmatyLocation)

	return dateInAlmaty.Year() == now.Year() &&
		dateInAlmaty.Month() == now.Month() &&
		dateInAlmaty.Day() == now.Day()
}

func ParseTimeOfDay(timeStr string) (time.Time, error) {
	return time.Parse("15:04", timeStr)
}
