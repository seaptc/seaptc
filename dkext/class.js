"use strict";

function formatClassNumber(n) {
  return n.toString().padStart(3, "0");
}

// descriptionValue returns the session event description for a class.
function descriptionValue(cls) {
  return `${formatClassNumber(cls.number)}: ${cls.title}${cls.titleNote ? ` (${cls.titleNote})` : ``}`
}

function escapeHTML(s) {
    let div = document.createElement('div');
    div.textContent = s.replace(/\s+/g, " ");
    return div.innerHTML;
}

// notesValue returns the session event notes for a class.
function notesValue(cls) {
  let titleNew = cls.titleNew
    ? `<span style="color: red">${escapeHTML(cls.titleNew)}</span> `
    : "";

  let titleNote = cls.titleNote
    ? ` (${escapeHTML(cls.titleNote)})`
    : "";

  let mostUsefulTo = "";
  if (cls.programs) {
    let parts = ["\n\n<p>This class is most useful to "];
    for (let i = 0; i < cls.programs.length; i++) {
      if (i === 0) {
        // no separator
      } else if (i === cls.programs.length - 1) {
        parts.push(" and ");
      } else {
        parts.push(", ");
      }
      parts.push(cls.programs[i]);
    }
    parts.push(".</p>");
    mostUsefulTo = parts.join("");
  }

  let session = "";
  if (cls.startSession && cls.endSession) {
    session = cls.startSession == cls.endSession
      ? `\n\n<p><i>(1 hour, session ${cls.startSession})</i></p>`
      : `\n\n<p><i>(${cls.endSession - cls.startSession + 1} hours, sessions ${cls.startSession} – ${cls.endSession})</i></p>`
  }

  return `<p><b>${formatClassNumber(cls.number)}: ${titleNew}${escapeHTML(cls.title)}${escapeHTML(titleNote)}</b></p>\n\n` +
    `<p>${escapeHTML(cls.description)}</p>${mostUsefulTo}${session}`;
}

// maxAttendeesValue converts the server"s capacity to DK"s max attendees.
//
//  Server: 0 is no limit, -1 is full
//  DK: blannk is no limit, 0 is full
function maxAttendeesValue(cls) {
  if (cls.capacity < 0) {
    return "0";
  } else if (cls.capacity > 0) {
    return cls.capacity.toString();
  } else {
    return ""
  }
}

// dateValue returns a data string given an array of integers with year, month
// and day.
function dateValue(t) {
  return `${t[1]}/${t[2]}/${t[0]}`;
}

function hourValue(t) {
  let hour = t[3];
  if (hour > 12) {
    hour -= 12;
  }
  return hour.toString();
}

function minuteValue(t) {
  return t[4].toString();
}

function ampmValue(t) {
  if (t[3] <= 12) {
    return "AM";
  }
  return "PM";
}


function controlValuesFromClass(cls) {
  let regBy = [cls.startTime[0],cls.startTime[1], cls.startTime[2], 9, 30];
  let regStart = [cls.startTime[0], 3, 1, 9, 30];
  let m = new Map();
  m.set("Description", descriptionValue(cls));
  m.set("Notes", notesValue(cls));
  m.set("MaxAttendees", maxAttendeesValue(cls));
  m.set("ActivityDate", dateValue(cls.startTime));
  m.set("ActivityFromHour", hourValue(cls.startTime));
  m.set("ActivityFromMin", minuteValue(cls.startTime));
  m.set("ActivityFromAMPM", ampmValue(cls.startTime));
  m.set("EndDate", dateValue(cls.endTime));
  m.set("ActivityTillHour", hourValue(cls.endTime));
  m.set("ActivityTillMin", minuteValue(cls.endTime));
  m.set("ActivityTillAMPM", ampmValue(cls.endTime));
  m.set("ContactEmail", "chiefseattleptc@gmail.com");
  m.set("Address", "9600 College Way North");
  m.set("City", "Seattle");
  m.set("State", "WA");
  m.set("Postal_Code", "98103");
  m.set("RegisterByDate", dateValue(regBy));
  m.set("RegisterByHour", hourValue(regBy));
  m.set("RegisterByMin", minuteValue(regBy));
  m.set("RegisterByAMPM", ampmValue(regBy));
  m.set("RegistrationStartDate", dateValue(regStart));
  m.set("RegistrationStartHour", hourValue(regStart));
  m.set("RegistrationStartMin", minuteValue(regStart));
  m.set("RegistrationStartAMPM", ampmValue(regStart));
  return m;
}
