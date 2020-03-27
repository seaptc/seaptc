// Content script for "Manage a Program Session".
"use strict";

function setup() {
  let title = document.querySelector("#pagetitlediv table tbody tr:first-of-type th");
  if (!title || (title.innerText.toUpperCase() !== "MANAGE A PROGRAM SESSION")) {
    console.log("DKE: could not find manage program session title on page");
    return;
  }

  let sp = new URLSearchParams(window.location.search);

  let classificationID = sp.get("classificationid");
  if (!classificationID) {
    classificationID = sp.get("emclassificationid");
  }
  if (!classificationID) {
    console.log("DKE: could not find classificationID");
    return;
  }

  let emActivityKey = sp.get("activitykey");
  if (!emActivityKey) {
    emActivityKey = sp.get("emactivitykey");
  }
  if (!emActivityKey) {
    console.log("DKE: could not find activityID");
    return;
  }

  // Replace ASP.NET post backs with URLs!!!
  let hrefs  = [];
  for (let e of document.querySelectorAll(".dk-three-dots")) {
    let tr = e.closest("tr")
    if (!tr) {
      continue
    }
    let activityKey = e.id;
    let href = window.location.origin +
      `/manageevents/confirmedit.asp` +
      `?classificationid=${classificationID}&activitykey=${activityKey}` +
      `&returntopage=%2fmanageevents%2feventsmanagement.aspx` +
      `%3femclassificationid%3d${classificationID}%26emactivitykey%3d${emActivityKey}`
    let anchors = tr.querySelectorAll("td a[title^=\"Edit \"]")
    // To enable debugging, we do not modify the last anchor (it's in the ... menu).
    for (let i = 0; i < anchors.length - 1; i++) {
      anchors[i].href = href;
    }
    hrefs.push(href)
  }

  // Tab explosion.
  let tbody = document.querySelector("#ActionMenu_ManageProgramSessions tbody")
  if (tbody) {
    let tr = tbody.appendChild(document.createElement("tr"))
    tr.appendChild(document.createElement("td")).textContent = "💥";
    let anchor = tr.appendChild(document.createElement("td")).appendChild(document.createElement("a"));
    anchor.textContent = "Tab Explosion";
    anchor.href = "#";
    anchor.style = "text-decoration: none;";
    anchor.onclick = () => {
      callBackground("createSessionEventTabs", hrefs).catch(reason => alert(reason));
      return false
    };
  }
}

setup();
