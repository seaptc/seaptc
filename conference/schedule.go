package conference

import (
	"fmt"
	"time"
)

type ScheduleTime struct {
	Start, End         time.Duration // minutes from start of day
	StartText, EndText string
}

func formatTime(h, m int) string {
	ampm := "AM"
	if h >= 12 {
		ampm = "PM"
		if h > 12 {
			h -= 12
		}
	}
	return fmt.Sprintf("%d:%02d %s", h, m, ampm)
}

func newScheduleTime(startHour, startMinute, endHour, endMinute int) *ScheduleTime {
	return &ScheduleTime{
		Start:     time.Duration(startHour)*time.Hour + time.Duration(startMinute)*time.Minute,
		End:       time.Duration(endHour)*time.Hour + time.Duration(endMinute)*time.Minute,
		StartText: formatTime(startHour, startMinute),
		EndText:   formatTime(endHour, endMinute),
	}
}

type ScheduleItem struct {
	*ScheduleTime
	Instructor  bool
	Description string
	Location    string
	Kind        string
	ClassNumber int
}

var (
	checkinScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(7, 40, 8, 15),
		Description:  "Check-in and Registration",
		Location:     "Wellness Center",
	}
	openingCermonyScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(8, 15, 8, 45),
		Description:  "Opening Ceremony",
		Location:     "Wellness Center",
	}
	break01ScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(10, 0, 10, 10),
		Description:  "Break – Visit the Midway or Scout Shop",
		Kind:         "break",
	}
	break12ScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(11, 10, 11, 20),
		Description:  "Break – Visit the Midway or Scout Shop",
		Kind:         "break",
	}
	break23ScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(1, 15, 1, 25),
		Description:  "Break – Visit the Midway or Scout Shop",
		Kind:         "break",
	}
	break34ScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(2, 25, 2, 35),
		Description:  "Break – Visit the Midway or Scout Shop",
		Kind:         "break",
	}
	break45ScheduleItem = &ScheduleItem{
		ScheduleTime: newScheduleTime(3, 35, 3, 45),
		Description:  "Break – Visit the Midway or Scout Shop",
		Kind:         "break",
	}
	SessionTimes = [NumSession]*ScheduleTime{
		newScheduleTime(9, 0, 10, 0),
		newScheduleTime(10, 10, 11, 10),
		newScheduleTime(11, 20, 13, 15),
		newScheduleTime(13, 25, 14, 25),
		newScheduleTime(14, 35, 15, 35),
		newScheduleTime(15, 45, 16, 45),
	}
	Seating1LunchTime = newScheduleTime(11, 10, 12, 15)
	Seating1ClassTime = newScheduleTime(12, 15, 13, 15)
	Seating2ClassTime = newScheduleTime(11, 20, 12, 20)
	Seating2LunchTime = newScheduleTime(12, 20, 13, 25)
)

func classScheduleItem(t *ScheduleTime, sc *SessionClass) *ScheduleItem {
	description := sc.Title
	if sc.Number != 0 {
		description = fmt.Sprintf("%d: %s%s", sc.Number, sc.ShortTitle(), sc.IofN())
	}
	return &ScheduleItem{
		ScheduleTime: t,
		Description:  description,
		Instructor:   sc.Instructor,
		Location:     sc.Location,
		ClassNumber:  sc.Number,
		Kind:         "session",
	}
}

func (conf *Conference) ParticipantSchedule(p *Participant) []*ScheduleItem {
	sessionClasses, lunch := conf.ParticipantSessionClassesAndLunch(p)

	schedule := []*ScheduleItem{
		checkinScheduleItem,
		openingCermonyScheduleItem,
		classScheduleItem(SessionTimes[0], sessionClasses[0]),
		break01ScheduleItem,
		classScheduleItem(SessionTimes[1], sessionClasses[1]),
		nil,
		nil,
		nil,
		classScheduleItem(SessionTimes[3], sessionClasses[3]),
		break34ScheduleItem,
		classScheduleItem(SessionTimes[4], sessionClasses[4]),
		break45ScheduleItem,
		classScheduleItem(SessionTimes[5], sessionClasses[5]),
	}

	lunchDescription := "Lunch"
	if p.LunchOption != "" {
		lunchDescription = fmt.Sprintf("Lunch: %s", p.LunchOption)
	}

	if lunch.Seating == 1 {
		schedule[5] = &ScheduleItem{
			ScheduleTime: Seating1LunchTime,
			Description:  lunchDescription,
			Location:     lunch.Location,
		}
		schedule[6] = classScheduleItem(Seating1ClassTime, sessionClasses[2])
		schedule[7] = break23ScheduleItem
	} else {
		schedule[5] = break12ScheduleItem
		schedule[6] = classScheduleItem(Seating2ClassTime, sessionClasses[2])
		schedule[7] = &ScheduleItem{
			ScheduleTime: Seating2LunchTime,
			Description:  lunchDescription,
			Location:     lunch.Location,
		}
	}
	return schedule
}
