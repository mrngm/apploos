package main

import (
	"time"
)

type ICalendar struct {
	Events []ICalendarEvent `xml:"vcalendar>components>vevent"`
}

type ICalendarEvent struct {
	//DateTime time.Time `xml:"dtstamp>date-time"`
	Summary     string `xml:"properties>summary>text"`
	StartTime   string `xml:"properties>dtstart>date-time"`
	StartTimeTZ string `xml:"properties>dtstart>parameters>tzid>text"`
	EndTime     string `xml:"properties>dtend>date-time"`
	EndTimeTZ   string `xml:"properties>dtend>parameters>tzid>text"`
	URL         string `xml:"properties>url>uri"`
	CostType    string `xml:"properties>x-cost-type>unknown"`
	TicketPrice string `xml:"properties>x-cost>unknown"`
	TicketURL   string `xml:"properties>x-tickets-url>unknown"`

	FullStartTime time.Time
	FullEndTime   time.Time
}
