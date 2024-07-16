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
}

h1, h2, h3 {
  font-size: 16pt;
  line-height: 1.5em; /* necessary for accurately doing magic with position: sticky-overlaps */
}

h2, h3 {
  padding-left: 2vw;
}
h3 {
  overflow-wrap: normal;
  overflow: hidden;
  text-wrap: nowrap;
}

.bg-green {
  background-color: #2ecc72 !important;
}
.day > section > h2 {
  background-color: #4169e1 !important;
}
.day > h1 {
  background-color: #d46a6a !important;
}
.day > h1 a {
    display: block;
    color: black !important;
    text-decoration: none;
}
.day > section > h2 a {
    display: block;
    color: black !important;
    text-decoration: none;
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
  background-color: #99b1f9 !important;
}
.roze.day > h1 {
  background-color: #ff10f0 !important;
}
/*
.roze .event:nth-of-type(even) {
  background-color: #fe5ef7 !important;
}
*/
.roze .event {
  background-color: #f200e8 !important;
}
.roze.day > section > h2 {
  background-color: #d600cc !important;
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
}
#main > section {
  height: 100vh;
  overflow-y: scroll;
  overflow-x: hidden;
  flex: 0 0 100vw;
  scroll-snap-align: center;
  scroll-snap-stop: always;
  max-width: 95vw;
}
#main > section:last-child {
  margin-right: 0em;
}
.event:has(h4:target) {
  box-shadow: 0 0 15px rgba(255, 215, 0, 0.7);
  background-color: #fffdf0 !important;
}

section > section {
  margin-bottom: 2ex;
  scroll-margin-top: 1.5em;
}

.artist {
  font-weight: bolder;
}

.event {
  padding: 0.75ex 0.75ex 1ex 0.5ex;
  font-size: 14pt;
  margin: 0.25em 0 0.25em 0;
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
}

label.hide {
  text-decoration: underline;
}

dd.description {
  display: none;
}

input:checked ~ dd.description {
  display: block;
}
input.meer-toggle {
  display: none;
}

label.hide::after {
  content: '(meer)';
}
input:checked ~ dd.summary label.hide::after {
  content: '(minder)';
}
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
  bottom: 0;
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
`)
var stylesheetCheckumShort = fmt.Sprintf("%x", sha256.Sum256(stylesheetCSS))[0:9]
