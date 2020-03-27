"use strict";

async function loadSettings() {
  let settings = await chromeStorageSync.get(defaultSettings);
  for (const name in settings) {
    const e = document.getElementById(name)
    e.value = settings[name];
    e.dataset.original = settings[name];
  }
}

async function saveSettings() {
  let settings = {}
  for (const name in defaultSettings) {
    settings[name] = document.getElementById(name).value
  }
  await chromeStorageSync.set(settings);
  for (const name in defaultSettings) {
    const e = document.getElementById(name);
    e.dataset.original = e.value;
  }
  document.getElementById("saveSettings").disabled = true;
}

function enableSaveButton() {
  for (const name in defaultSettings) {
    const e = document.getElementById(name);
    if (e.dataset.original !== e.value) {
      document.getElementById("saveSettings").disabled = false;
      return;
    }
  }
  document.getElementById("saveSettings").disabled = true;
  return false;
}

function setup() {
  loadSettings().catch(reason => alert(reason));
  
  document.getElementById("saveSettings").onclick = () => {
    saveSettings().catch(reason => alert(reason));
    return false;
  };

  for (const name in defaultSettings) {
    document.getElementById(name).oninput = enableSaveButton;
  }

  document.getElementById("upload").onclick = () => {
    callBackground("uploadRegistrations")
      .then(() => window.close())
      .catch(reason => alert(reason));
    return false;
  };

}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", setup);
} else {
  setup();
}
