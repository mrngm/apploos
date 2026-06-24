package main

import (
	"crypto/sha256"
	"fmt"
)

var scriptingJS = []byte(`
let storageAvailable = null;

function storageUsable() {
    console.log("storageUsable() called, storageAvailable: " + storageAvailable);
    if(storageAvailable !== null) {
        return storageAvailable;
    }

    // Carefully copy-pasted and slightly adapted from <https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API/Using_the_Web_Storage_API>
    let storage;
    try {
        storage = window.localStorage;
        const x = "__storage_test__";
        storage.setItem(x, x);
        storage.removeItem(x);
        storageAvailable = true;
    } catch (e) {
        storageAvailable = (
            e instanceof DOMException &&
            e.name === "QuotaExceededError" &&
            // acknowledge QuotaExceededError only if there's something already stored
            storage &&
            storage.length !== 0
        );
    }
    return storageAvailable;
}

function storeItem(name, value) {
    if(!storageUsable()) {
        return false;
    }

    try {
        window.localStorage.setItem(name, value)
    } catch (e) {
        if(e instanceof DOMException && e.name === "QuotaExceededError") {
            console.log("QuotaExceededError while trying to save " + name + " with value " + value);
        }
        return false;
    }

    return true;
}

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

function hideLocation(elem) {
    console.log("hideLocation for elem: " + elem.id + ", checked: " + elem.checked + ", classList: " + elem.classList);
    if(elem.parent) {
        elem.parent.scrollIntoView({block: "nearest"});
    }
    let locationClassId = null;
    elem.classList.forEach( (value, index, listObj) => {
        if(value.startsWith("hide-location-id-")) {
            locationClassId = value;
        }
    });
    if(locationClassId !== null) {
        console.log("Found location Class ID: " + locationClassId);
        let stored = storeItem(locationClassId, (elem.checked) ? "hidden" : "revealed");
        if(!stored) {
            console.log("Tried to store location hidden information for " + elem.id + ", but it failed");
        }
        triggerLocationHiddenToggle(locationClassId, elem.checked);
    }
}
function triggerLocationHiddenToggle(classId, isChecked) {
    const locationToggles = document.getElementsByClassName(classId);
    for(const toggle of locationToggles) {
        console.log("Toggling location " + toggle.id + " from " + toggle.checked + " to " + isChecked);
        toggle.checked = isChecked;
    }
}
function removeLocation(elem) {
    console.log("removeLocation for elem: " + elem.id + ", checked: " + elem.checked + ", classList: " + elem.classList);
    if(elem.parent) {
        elem.parent.scrollIntoView({block: "nearest"});
    }
    let locationClassId = null;
    elem.classList.forEach( (value, index, listObj) => {
        if(value.startsWith("remove-location-id-")) {
            locationClassId = value;
        }
    });
    if(locationClassId !== null) {
        console.log("Found location Class ID: " + locationClassId);
        let stored = storeItem(locationClassId, (elem.checked) ? "hidden" : "revealed");
        if(!stored) {
            console.log("Tried to store location hidden information for " + elem.id + ", but it failed");
        }
        triggerLocationRemovedToggle(locationClassId, elem.checked);
    }
}
function triggerLocationRemovedToggle(classId, isChecked) {
    const locationToggles = document.getElementsByClassName(classId);
    for(const toggle of locationToggles) {
        console.log("Removing location " + toggle.id + " from " + toggle.checked + " to " + isChecked);
        toggle.checked = isChecked;
    }
}
function toggleRemoveLocations() {
    const toggleCb = document.getElementById("eye-toggle-checkbox");
    const removeLocations = document.querySelectorAll(".remove-location-toggle");
    for(const elem of removeLocations) {
        let elemLabels = elem.labels;
        if(toggleCb.checked) {
            for(const elemLabel of elemLabels) {
                elemLabel.classList.remove("dont-display");
            }
            elem.parentNode.parentNode.classList.remove("dont-display");
        } else {
            for(const elemLabel of elemLabels) {
                elemLabel.classList.add("dont-display");
            }
            if(elem.checked) {
                elem.parentNode.parentNode.classList.add("dont-display");
            }
        }
    }
}

function syncStorageToPage() {
    if(!storageUsable()) {
        return;
    }
    for(let i = 0; i < window.localStorage.length; i++) {
        let storeKey = window.localStorage.key(i);
        if(storeKey.startsWith("hide-location-id-")) {
            let storeVal = window.localStorage.getItem(storeKey);
            console.log("Found location hide toggle in localStorage: " + storeKey + ": " + storeVal);
            if(storeVal === "hidden") {
                triggerLocationHiddenToggle(storeKey, true);
            }
            if(storeVal === "revealed") {
                triggerLocationHiddenToggle(storeKey, false);
            }
        }
        if(storeKey.startsWith("remove-location-id-")) {
            let storeVal = window.localStorage.getItem(storeKey);
            console.log("Found location removal toggle in localStorage: " + storeKey + ": " + storeVal);
            if(storeVal === "hidden") {
                triggerLocationRemovedToggle(storeKey, true);
            }
            if(storeVal === "revealed") {
                triggerLocationRemovedToggle(storeKey, false);
            }
        }
    }
    toggleRemoveLocations();
}
window.addEventListener("load", syncStorageToPage);
`)
var scriptingChecksumShort = fmt.Sprintf("%x", sha256.Sum256(scriptingJS))[0:9]
