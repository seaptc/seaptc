"use strict";

let sessionEventURLs = [];

async function createNextSessionEventTab() {
  const url = sessionEventURLs.pop();
  if (url) {
    await chromeTabs.create({ url: url });
  }
}

async function createSessionEventTabs(sender, urls) {
  sessionEventURLs = urls;
  await createNextSessionEventTab();
}

async function fetchClass(sender, number) {
  await createNextSessionEventTab();
  const settings = await chromeStorageSync.get(defaultSettings);
  const url = new URL(`/api/sessionEvents/${number}`, settings.server);
  const response = await fetch(url);
  const m = await response.json();
  if (m.error) {
    throw m.error;
  }
  return m.result;
}

async function uploadRegistrations(sender) {
  const settings = await chromeStorageSync.get(defaultSettings);
  const url = new URL("/dashboard/admin", settings.server).toString();
  const tabs = await chromeTabs.query({currentWindow: true, url: url});
  let tab = null;
  if (tabs.length > 0) {
    tab = tabs[0];
    await chromeTabs.highlight({windowId: tab.windowId, tabs: [tab.index]});
  } else {
    tab = await chromeTabs.create({ url: url });
  }
  const cookies = await chromeCookies.getAll({url: "https://seattlebsa.doubleknot.com"});
  const cookieHeader = cookies.map(c => `${c.name}=${c.value}`).join("; ");
  await chromeTabs.executeScript(tab.id, {
    code: `document.getElementById("dkCookies").value = ${JSON.stringify(cookieHeader)}; document.getElementById("fetchRegistrations").click();`
  });
}

listen({
  "createSessionEventTabs": createSessionEventTabs,
  "fetchClass": fetchClass,
  "uploadRegistrations": uploadRegistrations,
});
