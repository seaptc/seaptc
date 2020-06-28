// Content script for "Create and Modify a Session Event".
"use strict";

var form = null;   // Look for controls in this form.
var info = null;   // Display information in this element.

// fetchClass fetches class with given number from the server and updates the
// contents of the info element as appropriate.
async function fetchClass(num) {
  showMessage(`Fetching class ${num} from the server...`);
  let [err, cls] = await catchEm(callBackground("fetchClass", num));
  if (err) {
    showMessage(err);
  } else {
    showProposedModifications(controlValuesFromClass(cls));
  }
}

// showMessage replaces the info element contents with the given message and
// the class number button.
function showMessage(message) {
  while (info.firstChild) {
    info.removeChild(info.firstChild);
  }
  info.appendChild(createClassNumberButton());
  if (message) {
    info.appendChild(document.createElement("div")).textContent = message;
  }
}

function getControl(name) {
    // Special cases:
    //  - the Notes <textarea> is hidden by CKEdit, so allow that to be hidden.
    //  - The name of the max attendees field changes depending on registrant
    //    type charges for the session event. Look for both names.

    let control = form.querySelector(`[name="${name}"]`);

    if (control && (control.type !== "hidden" || name === "Notes")) {
      return control;
    }
    if (name == "MaxAttendees") {
      control = form.querySelector(`[name="MaxEventAttendees"]`);
      if (control && control.type != "hidden") {
        return control;
      }
    }
    return nil;
}

// showProposedModifications replaces the info element content with button to save
// the changes and a definition list with the proposed changes (if any).
function showProposedModifications(values) {
  let mods = document.createElement("div");
  let notFound = [];

  for (let [name, value] of values) {
    let control = getControl(name);
    if (!control) {
      notFound.push(name);
      continue;
    }

    // DK replaces &amp; with & in the Notes field. Do the same here to avoid
    // noise in the output.
    if (name == "Notes") {
      value = value.replace(/&amp;/g, "&");
    }

    if (control.value !== value) {
      if (control.value.toString().length < 30 && value.toString().length < 30) {
        mods.appendChild(document.createElement("div")).textContent = `${name}: ${control.value} → ${value}`
      } else {
        mods.appendChild(document.createElement("div")).textContent = `${name}:`
        const style ="padding-left: 2em; text-indent: -1.25em; max-width: 800px;";
        if (control.value) {
          let d = mods.appendChild(document.createElement("div"))
          d.style = style
          d.textContent = `← ${control.value}`
        }
        if (value) {
          let d = mods.appendChild(document.createElement("div"))
          d.style = style
          d.textContent = `→ ${value}`
         }
      }
    }
  }

  while (info.firstChild) {
    info.removeChild(info.firstChild);
  }
  info.appendChild(createSaveButton(values, "Set and Save", true));
  info.appendChild(createSaveButton(values, "Set", false));
  info.appendChild(createClassNumberButton());

  if (notFound.length) {
    info.appendChild(document.createElement("div")).textContent = `Missing or hidden controls: ${notFound.join(", ")}`;
  }

  if (mods.hasChildNodes()) {
    info.appendChild(mods);
  } else {
    info.appendChild(document.createElement("div")).textContent = "The session event is up-to-date.";
  }
}

// setControlValues sets the given values on the form's controls.
function setControlValues(values) {
  for (let [name, value] of values) {
    if (name === "Notes") {
      let script = document.createElement("script")
      script.textContent = `CKEDITOR.instances["Notes"].setData(${JSON.stringify(value)})`
      document.head.appendChild(script);
      script.remove();
      continue
    }
    let control = getControl(name);
    if (!control) {
      console.log(`DKE: control not found or hidden: ${name}`)
      continue;
    }
    control.value = value;
  }
}

// createClassNumberButton creates a button with a click handler that prompts
// for and loads a class.
function createClassNumberButton() {
  let b = document.createElement("button");
  b.textContent = "Class number";
  b.style = "margin: 5px;"
  b.onclick = (e) => {
    let num = window.prompt("Class number", getClassNumber());
    if (num) {
      fetchClass(num);
    }
    return false;
  };
  return b;
}

// createSaveButton creates a button that sets the given values on the controls
// and optionally clicks the form's save button.
function createSaveButton(values, name, click) {
  let b = document.createElement("button");
  b.textContent = name;
  b.style = "margin: 5px;"
  b.onclick = (e) => {
    setControlValues(values);
    let b = form.querySelector("#iSaveEditEventButton");
    if (!b) {
      console.log("could not find save button");
      return false;
    }
    if (click) {
      // Delay click to give CKEdit an opportunity to run.
      setTimeout(() => { b.click(); }, 100);
    }
    return false;
  };
  return b;
}

// getClassNumber gets the class number from the description control or returns
// "" if the control is not found or empty.
function getClassNumber() {
  let description = form.querySelector("[name=\"Description\"]");
  if (!description) {
    console.log("DKE: could not find description control");
    return "";
  }
  let s = description.value;
  if (!s.match(/^\d\d\d:/)) {
    return "";
  }
  return s.substr(0, 3);
}

function setup() {
  let title = document.querySelector("#pagetitlediv table tbody tr:first-of-type th");
  if (!title ||
      (
        (title.innerText.toUpperCase() !== "MODIFY ALL OCCURRENCES OF A SESSION EVENT") &&
        (title.innerText.toUpperCase() !== "CREATE AND MODIFY A SESSION EVENT") &&
        (title.innerText.toUpperCase() !== "CREATE ALL OCCURRENCES OF A SESSION EVENT")
      )) {
    console.log("DKE: could not find session event title on page");
    return;
  }

  form = document.querySelector("form[action=\"UpdateActivity.asp\"]");
  if (!form) {
    console.log("DKE: could not find activity form");
    return;
  }

  let tbody = form.querySelector("table > tbody");
  if (!tbody) {
    console.log("DKE: could not find form > table > tbody");
    return
  }

  let tr = tbody.insertBefore(document.createElement("tr"), tbody.firstChild);
  info = tr.appendChild(document.createElement("td"));
  info.colspan = 2;

  let num = getClassNumber();
  if (num) {
    fetchClass(num);
  } else {
    showMessage("");
  }

}

setup();
