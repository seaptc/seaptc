"use strict";

function catchEm(promise) {
  return promise.then(value => [null, value], reason => [reason]);
}

function getTrap(target, property) {
  return function(...args) {
    return new Promise(function(resolve, reject) {
      target[property](...args, function(value) {
        if (chrome.runtime.lastError) {
          reject(chrome.runtime.lastError);
        } else {
          resolve(value);
        }
      })
    })
  }
}

function wrapext(path) {
  let target = chrome;
  for (name of path.split(".")) {
    target = target[name];
    if (target === undefined) {
      return undefined;
    }
  }
  return new Proxy(target, {get: getTrap});
}

const chromeTabs = wrapext("tabs");
const chromeStorageSync = wrapext("storage.sync");
const chromeCookies = wrapext("cookies");

function callBackground(...nameAndArgs) {
  return new Promise((resolve, reject) => {
    chrome.runtime.sendMessage(nameAndArgs, response => {
      if (chrome.runtime.lastError) {
        reject(chrome.runtime.lastError)
      } else {
        let [reason, value] = response;
        if (reason !== null) {
          reject(reason);
        } else {
          resolve(value);
        }
      }
    });
  });
}

function callTab(tabID, ...nameAndArgs) {
  return new Promise((resolve, reject) => {
    chrome.tabs.sendMessage(tabID, nameAndArgs, response => {
      if (chrome.runtime.lastError) {
        reject(chrome.runtime.lastError)
      } else {
        let [reason, value] = response;
        if (reason !== null) {
          reject(reason);
        } else {
          resolve(value);
        }
      }
    });
  });
}

function listen(handlers) {
  chrome.runtime.onMessage.addListener(([name, ...args], sender, sendResponse) => {
    const handler = handlers[name];
    if (!handler) {
      sendResponse([`No handler for ${name}`]);
      return false;
    }
    Promise.resolve(handler(sender, ...args)).then(
      value => sendResponse([null, value]),
      reason => {
        console.log(reason);
        if (reason.message) {
          reason = reason.message;
        }
        sendResponse([reason])
      });
    return true;
  });
}

const defaultSettings = {
  server: "https://seaptc.org"
}
