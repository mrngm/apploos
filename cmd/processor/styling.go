package main

import (
	"crypto/sha256"
	"fmt"
)

var stylesheetCSS = []byte(`
* {
  /* reset ALL the things */
  margin: 0;
  padding: 0;
  font-family: sans-serif;
  box-sizing: border-box;
}

body {
  background-color: #f4f4f4;
  color: black;
  display: flex;
  flex-direction: column;
  height: 100vh;
  overscroll-behavior: none;
}

h1, h2, h3 {
  font-size: 14pt;
  line-height: 1.5em; /* necessary for accurately doing magic with position: sticky-overlaps */
}

h2, h3 {
  padding-left: 2vw;
}
h3, a.location-title, h1 {
  overflow-wrap: normal;
  overflow: hidden;
  text-wrap: nowrap;
}

.bg-green {
  background-color: #2ecc72 !important;
}
.day > section > h2 {
  /* hsl: 205 / 100% / 52% */
  background-color: #0A99FF !important;
}
.day > h1 {
  /* hsl: 205 / 100% / 70% */
  background-color: #66BFFF !important;
}
.day > h1 a {
    /* display: block; */
    color: black !important;
    text-decoration: none;
}
.day > section > h2 a {
    /* display: block; */
    color: black !important;
    text-decoration: none;
}
a.location-title {
    padding-left: 1ex;
}
.bg-yellow {
  background-color: #ffd700 !important;
}
/*
.event:nth-of-type(even) {
  background-color: #c8d6fe !important;
}
*/
.event {
  /* hsl: 205 / 100% / 63% */
  background-color: #42B0FF !important;
}
.roze.day > h1 {
  /* hsl: 307 / 100% / 80% */
  background-color: #FF99F8 !important;
  /* hsl: 307 / 100% / 62% */
  background-color: #FF3DE8 !important;
}
/*
.roze .event:nth-of-type(even) {
  background-color: #fe5ef7 !important;
}
*/
.roze .event {
  /* hsl: 307 / 100% / 73% */
  background-color: #FF75EF !important;
  /* hsl: 307 / 100% / 80% */
  background-color: #FF99F8 !important;
  /* hsl: 307 / 100% / 90% */
  background-color: #FFCCFC !important;
}
.roze.day > section > h2 {
  /* hsl: 307 / 100% / 62% */
  background-color: #FF3DE8 !important;
  /* hsl: 307 / 100% / 73% */
  background-color: #FF75EF !important;
}
.roze > section > h3 {
  background-color: #fe6ef7 !important;
}

.roze .event.past {
    filter: saturate(200%);
    background-color: #f25eec !important;
    color: #7f7e7e !important;
}

.fg-white {
  color: white !important;
}

.horizontal-filler {
  min-height: 200px;
}

.sticky-0 {
  position: sticky;
  top: 0;
  background: white;
  z-index: 100;
  text-align: center;
}
.sticky-1 {
  position: sticky;
  top: 1.5em;
  background: white;
  z-index: 80;
}
.sticky-2 {
  position: sticky;
  top: 3em;
  background: white;
  z-index: 60;
}

#main {
  min-height: 5vh;
  margin: auto;
  flex: 1;
  display: flex;
  overflow-x: scroll;
  width: 100vw;
  scroll-snap-type: x mandatory;
  scroll-behavior: smooth;
  padding: 0 2.5vw;
  gap: 0.5em;
  margin-bottom: 2.125em;
}
#main > section {
  height: stretch;
  overflow-y: scroll;
  overflow-x: hidden;
  flex: 0 0 100vw;
  scroll-snap-align: center;
  scroll-snap-stop: always;
  max-width: 95vw;
}
#main > section:first-child {
  margin-left: 0em;
}
#main > section:last-child {
  margin-right: 0em;
}
.event:has(h4:target) {
  box-shadow: 0 0 15px rgba(255, 215, 0, 0.7);
  background-color: #fffdf0 !important;
}

section > section {
  scroll-margin-top: 1.75em;
}

.artist {
  font-weight: bolder;
}

.event {
  padding: 0.25ex 0.75ex 0.25ex 0.5ex;
  font-size: 12pt;
  margin: 0.125em 0 0.125em 0;
}
.event:first-child {
  margin-top: 0.25em;
}
.event:last-child {
  margin-bottom: 0.5em;
}
.event:only-child {
  margin-top: 0;
  margin-bottom: 0;
}

/* event after h3, subsequent-sibling combinator */
h3 ~ .event > h4 {
  scroll-margin-top: 5.5em;
}
.event > h4 {
  scroll-margin-top: 4em;
}

dt {
}
dd {
}
h3 ~ dt {
  padding-left: 2.5ex;
}

.summary {
  font-style: italic;
  text-align: justify;
  line-height: 1.5;
  hyphens: auto;
}
.description {
  text-align: justify;
  line-height: 1.5;
  hyphens: auto;
  white-space: pre-wrap;
}

dd.description {
  display: none;
}

/* Toggle descriptions */
input.hide-description-toggle:checked ~ dd.description {
  display: block;
}
input.hide-description-toggle {
  display: none;
}

label.hide-description {
  text-decoration: underline;
}
label.hide-description::after {
  content: '(meer)';
}
input.hide-description-toggle:checked ~ dd.summary label.hide-description::after {
  content: '(minder)';
}
/* End toggle description */

/* Toggle location; default state is unchecked, and we called it 'hide-location', so only hide the location when it's checked  */
section:has(h2 > input.hide-location-toggle:checked) {
  margin-bottom: 0;
}
section:last-child:has(h2 > input.hide-location-toggle:checked) {
  margin-bottom: 0.5em;
}
h2:has(input.hide-location-toggle:checked) ~ div.events-all {
  display: none;
}
h2:has(input.hide-location-toggle:not(:checked)) ~ div.events-all {
  display: block;
}
input.hide-location-toggle {
  display: none;
}

label.hide-location {
  padding-left: 0.5em;
  padding-right: 0.5em;
  /* hsl: 205 / 100% / 70% */
  background-color: #66BFFF;
  user-select: none;
}
.roze > section > h2 label.hide-location {
  /* hsl: 307 / 100% / 80% */
  background-color: #FF99F8 !important;
  /* hsl: 307 / 100% / 62% */
  background-color: #FF3DE8 !important;
}
label.hide-location::after {
  content: '\2716'; /* -- */
  content: '\25bd'; /* -- */
  content: '\23fc'; /* -- */
  content: '   \2212'; /* -- */
}
input.hide-location-toggle:checked ~ label.hide-location::after {
  content: '\271a'; /* + */
  content: '\25c1'; /* + */
  content: '\002b'; /* + */
}
/* End location description */

/* Magic CSS to hide/show based on target click
.show, .hide:target, dd.description {
  display: none;
}
.hide:target + .show, .hide:target ~ dd.description {
  display: block;
}
dd.summary ~ a.hide, dd.summary ~ a.show {
  padding-left: 2.5ex;
}
*/

#testing-banner {
  position: fixed;
  min-height: 1.5em;
  font-size: 12pt;
  background-color: orange;
  color: white;
  font-variant: bold;
  bottom: 1.5em;
  left: 0;
  z-index: 120;
  padding: 0.5ex;
}
.event.now {
    background-color: #71f241 !important;
}
.event.past {
    filter: saturate(50%);
    background-color: #c8d6fe !important;
    color: #7f7e7e !important;
}

.fire-text {
  text-shadow: 0ex -0.1ex 0.2ex #fff, 0ex -0.1ex 0.5ex #FF3, 0ex -0.5ex 1ex #F90, 0ex -1ex 2ex #C33;
}

div#nav {
  position: fixed;
  bottom: 0;
  right: 0;
  z-index: 150;
  width: 100vw;
  font-size: 16pt;
  background-color: white;
}

ul.navigation {
  display: flex;
  flex-flow: row wrap;
  list-style-type: none;
}
ul.navigation li {
  flex: 1 1 20vw;
  text-align: center;
  margin-top: .2ex;
  margin-left: .2ex;
  margin-right: .2ex;
}
ul.navigation li:first-child {
  margin-left: 0px !important;
}
ul.navigation li:last-child {
  margin-right: 0px !important;
}
li button {
  font-size: 14pt;
  background-color: #4169e1;
  text-align: center;
  text-decoration: none;
  height: 100%;
  width: 100%;
  padding: .1ex;
  border-top: 2px solid #6a89ea;
  border-right: 2px solid #38497e;
  border-bottom: 2px solid #38497e;
  border-left: 2px solid #6a89ea;
  color: white;
}
button {
  touch-action: manipulation;
  user-select: none;
}

li button:active {
  background-color: #99b1f9;
}
`)
var stylesheetCheckumShort = fmt.Sprintf("%x", sha256.Sum256(stylesheetCSS))[0:9]
