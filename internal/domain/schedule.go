package domain

import (
	"time"
)

type Schedule struct {
	ID           int64     `json:"id"`
	SpecialistID int64     `json:"specialist_id"`
	DayOfWeek    int       `json:"day_of_week"`
	StartTime    string    `json:"start_time"`
	EndTime      string    `json:"end_time"`
	SlotTime     int       `json:"slot_time"`
	ExcludeTimes []string  `json:"exclude_times"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type WorkTimeSlot struct {
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

type TimeSlot struct {
	Time      string `json:"time"`
	Available bool   `json:"available"`
}

type DaySchedule struct {
	WorkTime []WorkTimeSlot `json:"work_time"`
}

type WeekSchedule struct {
	Monday    *DaySchedule `json:"monday,omitempty"`
	Tuesday   *DaySchedule `json:"tuesday,omitempty"`
	Wednesday *DaySchedule `json:"wednesday,omitempty"`
	Thursday  *DaySchedule `json:"thursday,omitempty"`
	Friday    *DaySchedule `json:"friday,omitempty"`
	Saturday  *DaySchedule `json:"saturday,omitempty"`
	Sunday    *DaySchedule `json:"sunday,omitempty"`
}

type CreateScheduleDTO struct {
	WeekSchedule WeekSchedule `json:"week_schedule" binding:"required"`
	SlotTime     int          `json:"slot_time" binding:"required"`
}

type UpdateScheduleDTO struct {
	WeekSchedule WeekSchedule `json:"week_schedule" binding:"required"`
	SlotTime     *int         `json:"slot_time,omitempty"`
}

type ScheduleFilter struct {
	SpecialistID *int64 `json:"specialist_id"`
	DayOfWeek    *int   `json:"day_of_week"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}
