package main

var testingBanner = `<div id="testing-banner">TESTOMGEVING, <a href="https://apploos.nl/4df/">klik hier</a> om naar de live website te gaan.</div>`

var navigation = `<div id="nav">
      <ul class="navigation">
        <li class="nav-left"><button onclick="left()">D-</button></li>
        <li class="nav-up"><button onclick="up()">L-</button></li>
        <li class="nav-down"><button onclick="down()">L+</button></li>
        <li class="nav-right"><button onclick="right()">D+</button></li>
      </ul>
    </div>`

var htmlTemplate = `<!DOCTYPE html>
<html lang="nl">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width" />
    <title>Vierdaagsefeesten 2025</title>
    <link rel="stylesheet" type="text/css" href="style.css?{{ .StylesheetChecksumShort }}" />
    <script type="text/javascript">
        function scrollToAnchorOrDay() {
            if(location.hash != "") {
                let el = document.getElementById(location.hash.substring(1));
                if(el != null) {
                    el.scrollIntoView();
                }
            } else {
                let today = new Date();
                    if(today.getFullYear() == 2025 && today.getMonth() + 1 == 7) {
                    let currentVierdaagseDay = today.getDate() - 11;
                    if(currentVierdaagseDay >= 1 && currentVierdaagseDay <= 7) {
                        let el = document.getElementById('day-' + currentVierdaagseDay);
                        el.scrollIntoView();
                    }
                }
            }
        }
        window.addEventListener("load", scrollToAnchorOrDay);

        function up() {
            let firstElement = null;
            const locations = document.querySelectorAll(".location-title")
            for(const el of locations) {
                if(!isInViewport(el)) {
                    continue;
                }
                firstElement = el;
                break;
            }
            let previousSection = firstElement.parentNode.parentNode.previousElementSibling;
            if(previousSection != null) {
                previousSection.scrollIntoView();
            }
        }
        function down() {
            let firstElement = null;
            const locations = document.querySelectorAll(".location-title")
            for(const el of locations) {
                if(!isInViewport(el)) {
                    continue;
                }
                firstElement = el;
                break;
            }
            let nextSection = firstElement.parentNode.parentNode.nextElementSibling;
            if(nextSection != null) {
                nextSection.scrollIntoView();
            }
        }
        function left() {
            let firstElement = null;
            const days = document.querySelectorAll(".day")
            for(const el of days) {
                if(!isInViewport(el)) {
                    continue;
                }
                firstElement = el;
                break;
            }
            let previousSection = firstElement.previousElementSibling;
            if(previousSection != null) {
                previousSection.scrollIntoView();
            }
        }
        function right() {
            let firstElement = null;
            const days = document.querySelectorAll(".day")
            for(const el of days) {
                if(!isInViewport(el)) {
                    continue;
                }
                firstElement = el;
                break;
            }
            let nextSection = firstElement.nextElementSibling;
            if(nextSection != null) {
                nextSection.scrollIntoView();
            }
        }
    </script>
  </head>
  <body>
    <a name="top"></a>
    {{- $isProd := .IsProduction -}}
    {{ if not .IsProduction }} {{- .TestingBanner -}} {{ end }}
    {{ if not .IsProduction }} {{- .Navigation -}} {{ end }}
    <div id="main" class="container">
    {{- $schedule := .Schedule -}}
    {{- range $index, $day := $schedule.Days -}}
      {{- $currentDayNumber := add $index 1 -}}
      {{- $currentDayId := $day.Id }}
      <section class="{{ if isRoze $day.Date -}} roze {{ end }}day" id="day-{{ $currentDayNumber }}">
        <h1 class="sticky-0"><a href="#day-{{ $currentDayNumber }}">Dag {{ $currentDayNumber }}, {{ if isRoze $day.Date -}} Roze {{ end }}<time {{ formatRFC3339DatetimeAttr $day.Date }}>{{ $day.Title }}</time></a></h1>
      {{- range $location := $schedule.Locations -}}
        {{- $nrProgsToday := index $location.TotalNrProgramsByDay $currentDayId -}}
        {{- if ne $nrProgsToday 0 }}
        <section id="day-{{ $currentDayNumber }}-lokatie-{{ $location.Slug }}">
          <h2 class="sticky-1">{{- if not $isProd -}}<input type="checkbox" class="hide-location-toggle" id="hide-location-{{ $currentDayId }}-{{ $location.Id }}" /> <label for="hide-location-{{ $currentDayId }}-{{ $location.Id }}" class="hide-location"></label>{{- end -}} <a class="location-title" href="#day-{{ $currentDayNumber }}-lokatie-{{ $location.Slug }}">{{ $location.Title }}</a></h2>
          <div class="events-all">
        {{- range $dayId, $progs := $location.ProgramsByDay -}}
          {{- if ne $dayId $currentDayId -}} {{- continue -}} {{- else -}}
            {{- range $prog := $progs -}}
              {{- if ne $prog.LocationId $location.Id -}} {{- continue -}} {{- else -}}
                {{/* These are programs on the "main" location */}}
            <div class="event">
              <h4 id="{{ $prog.Slug }}"><time {{ formatRFC3339DatetimeAttr $prog.FullStartTime }}>{{ formatHourMins $prog.FullStartTime }}{{- if $prog.StartTimeEstimated -}}?{{- end -}}</time> - <time {{ formatRFC3339DatetimeAttr $prog.FullEndTime }}>{{ formatHourMins $prog.FullEndTime }}{{- if $prog.EndTimeEstimated -}}?{{- end -}}</time> {{ $prog.Title }}{{- if decimalGtZero $prog.TicketPrice }} ({{- if len $prog.TicketLink | ne 0 -}}<a href="{{- $prog.TicketLink -}}" target="_blank">€</a>{{- else -}}€{{- end -}}) {{- if $prog.TicketsSoldOut }} (uitverkocht) {{- end -}}{{- end -}}</h4>
                {{- $progDetailsEmpty := len $prog.Details | eq 0 -}}
                {{- if eq $prog.Title $prog.Details | or $progDetailsEmpty -}}
              <dd class="summary">{{ $prog.Summary }}</dd>
                {{- else -}}
              <input type="checkbox" class="hide-description-toggle" id="hide-description-{{ $prog.Id }}" />
              <dd class="summary">{{ $prog.Summary }} <label for="hide-description-{{ $prog.Id }}" class="hide-description"></label></dd>
              <dd class="description">{{ $prog.Details }} </dd>
                {{- end }}
            </div>
              {{- end -}}
            {{- end -}}
          {{- end -}}
          {{- range $childLocation := $location.Children -}}
            {{- $nrProgsToday := index $childLocation.TotalNrProgramsByDay $currentDayId -}}
            {{- if ne $nrProgsToday 0 }}
            <h3 class="sticky-2" id="day-{{ $currentDayNumber }}-lokatie-{{ $location.Slug }}-{{ $childLocation.Slug }}">{{ $childLocation.Title }}</h3>
            <div class="events-child-location">
            {{- range $dayId, $progs := $childLocation.ProgramsByDay -}}
              {{- if ne $dayId $currentDayId -}} {{- continue -}} {{- else }}
              {{- range $prog := $progs -}}
                {{- if ne $prog.LocationId $childLocation.Id -}} {{- continue -}} {{- else -}}
                  {{/* These are programs on a child location of the main location, such as separate stages or rooms */}}
            <div class="event">
              <h4 id="{{ $prog.Slug }}"><time {{ formatRFC3339DatetimeAttr $prog.FullStartTime }}>{{ formatHourMins $prog.FullStartTime }}{{- if $prog.StartTimeEstimated -}}?{{- end -}}</time> - <time {{ formatRFC3339DatetimeAttr $prog.FullEndTime }}>{{ formatHourMins $prog.FullEndTime }}{{- if $prog.EndTimeEstimated -}}?{{- end -}}</time> {{ $prog.Title }}{{- if decimalGtZero $prog.TicketPrice }} ({{- if len $prog.TicketLink | ne 0 -}}<a href="{{- $prog.TicketLink -}}" target="_blank">€</a>{{- else -}}€{{- end -}}) {{- if $prog.TicketsSoldOut }} (uitverkocht) {{- end -}}{{- end -}}</h4>
                  {{- $progDetailsEmpty := len $prog.Details | eq 0 -}}
                  {{- if eq $prog.Title $prog.Details | or $progDetailsEmpty -}}
              <dd class="summary">{{ $prog.Summary }}</dd>
                  {{- else -}}
              <input type="checkbox" class="hide-description-toggle" id="hide-description-{{ $prog.Id }}" /><dd class="summary">{{ $prog.Summary }} <label for="hide-description-{{ $prog.Id }}" class="hide-description"></label></dd>
              <dd class="description">{{ $prog.Details }} </dd>
                  {{- end }}
            </div>
                {{- end -}}
              {{- end -}}
            </div>
            {{- end -}}
            {{- end -}}
          {{- end -}}
          {{- end -}}
        {{- end }}
          </div>
        </section>
        {{- end -}}
      {{- end }}
      </section>
      {{- end -}}
    </div>
    <script type="text/javascript">
    // From https://www.javascripttutorial.net/dom/css/check-if-an-element-is-visible-in-the-viewport/
    function isInViewport(el) {
        const rect = el.getBoundingClientRect();
        return (
            rect.top >= 0 &&
            rect.left >= 0 &&
            rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
            rect.right <= (window.innerWidth || document.documentElement.clientWidth)

        );
    }

    function highlightNow() {
      let now = new Date()
      document.querySelectorAll(".event").forEach(x => {
        const [start, end] = Array.from(x.querySelectorAll("time")).map(y => new Date(y.getAttribute("datetime")))
        if (start <= now && now <= end) {
          x.classList.add("now")
        } else if (end <= now) {
          x.classList.add("past")
          x.classList.remove("now")
        }
      })
      now = new Date();
      const nextMinute = new Date(now.getFullYear(), now.getMonth(), now.getDate(), now.getHours(), now.getMinutes() + 1, 0, 0);
      setTimeout(highlightNow, nextMinute - now);
    }
    highlightNow()
    </script>
  </body>
</html>
`

// vim: et:sw=2:ts=2:
