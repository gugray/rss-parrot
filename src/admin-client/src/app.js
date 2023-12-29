import {feedUrls} from "./urls-smallweb.js";
// import {feedUrls} from "./urls-debug.js";

const elmTxtServerUrl = document.getElementById("txtServerUrl");
const elmTxtApiKey = document.getElementById("txtApiKey");
const elmTxtImportParallelReqs = document.getElementById("txtImportParallelReqs");
const elmBtnImportFeeds = document.getElementById("btnImportFeeds");
const elmLblFeedCount = document.getElementById("lblFeedCount");
const elmLblFeedsRequested = document.getElementById("lblFeedsRequested");
const elmLblFeedsFailed = document.getElementById("lblFeedsFailed");
const elmLblReqPerSec = document.getElementById("lblReqPerSec");
const elmLblActiveReqs = document.getElementById("lblActiveReqs");
const elmPnlImportErrors = document.getElementById("pnlImportErrors");

const stgsKey = "rss-parrot-admin-params";
const postFeedTimeoutSec = 60;
const reqStatPeriodSec = 10;

const feedImportStats = {
  nRequested: 0,
  reqLog: [],
  nActiveReqs: 0,
  nMaxParallel: 1,
  running: false,
  canceled: false,
  feedIx: 0,
  nFailed: 0,
  setCanceled: (canceled) => {
    feedImportStats.canceled = canceled;
  },
};

initParams();
initFeedImport();

function initParams() {
  let stgs = null;
  let str = localStorage.getItem(stgsKey);
  if (str) {
    try { stgs = JSON.parse(str); }
    catch {}
  }
  if (stgs) {
    elmTxtServerUrl.value = stgs.url;
    elmTxtApiKey.value = stgs.api_key;
  }
  const saveSettings = () => {
    const stgs = {
      url: elmTxtServerUrl.value,
      api_key: elmTxtApiKey.value,
    };
    localStorage.setItem(stgsKey, JSON.stringify(stgs));
  };
  elmTxtServerUrl.addEventListener("input", () => saveSettings());
  elmTxtApiKey.addEventListener("input", () => saveSettings());
}

function initFeedImport() {
  resetFeedImportStats();
  updateFeedImportValues();
  elmBtnImportFeeds.addEventListener("click", startStopFeedImport);
}

function resetFeedImportStats() {
  feedImportStats.nRequested = 0;
  feedImportStats.reqLog = [];
  feedImportStats.nActiveReqs = 0;
  feedImportStats.feedIx = 0;
  feedImportStats.nFailed = 0;
  feedImportStats.nMaxParallel = Number.parseInt(elmTxtImportParallelReqs.value);
}

function updateFeedImportValues() {

  elmLblFeedCount.innerText = feedUrls.length;
  elmLblFeedsRequested.innerText = feedImportStats.nRequested;
  elmLblFeedsFailed.innerText = feedImportStats.nFailed;
  elmLblActiveReqs.innerText = feedImportStats.nActiveReqs;

  const periodStart = new Date(Date.now() - reqStatPeriodSec * 1000);
  let nCompletedInPeriod = 0;
  for (const req of feedImportStats.reqLog) {
    if (req.endTime > periodStart) ++nCompletedInPeriod;
  }
  const reqPerSec = nCompletedInPeriod / reqStatPeriodSec;
  elmLblReqPerSec.innerText = reqPerSec.toFixed(2);

  if (feedImportStats.running) {
    setTimeout(updateFeedImportValues, 1000);
  }
}

function startStopFeedImport() {
  if (!feedImportStats.running) {
    resetFeedImportStats();
    feedImportStats.running = true;
    feedImportStats.setCanceled(false);
    elmBtnImportFeeds.innerText = "Stop";
    elmPnlImportErrors.innerText = "";
    void feedImportFun();
    updateFeedImportValues();
  }
  else {
    feedImportStats.setCanceled(true);
  }
}

async function feedImportFun() {

  function fireRequests() {
    while (feedImportStats.nActiveReqs < feedImportStats.nMaxParallel) {
      if (feedImportStats.feedIx >= feedUrls.length) break;
      if (feedImportStats.canceled) break;
      const feedUrl = feedUrls[feedImportStats.feedIx];
      ++feedImportStats.feedIx;
      ++feedImportStats.nActiveReqs;
      const logItem = {startTime: Date.now()};
      feedImportStats.reqLog.push(logItem);
      void callImportFeed(feedUrl, (err) => {
        --feedImportStats.nActiveReqs;
        ++feedImportStats.nRequested;
        logItem.endTime = Date.now();
        if (err) {
          ++feedImportStats.nFailed;
          elmPnlImportErrors.innerText += feedUrl + " => " + err + "\n";
        }
        fireRequests();
      });
    }
    if (feedImportStats.nActiveReqs == 0) {
      feedImportStats.running = false;
      elmBtnImportFeeds.innerText = "Start";
    }
  }
  fireRequests();
}

async function callImportFeed(siteUrl, done) {

  let finished = false;
  const abortCtrl = new AbortController();
  setTimeout(() => {
    if (!finished) abortCtrl.abort("Request has timed out");
  }, 1000 * postFeedTimeoutSec);

  const baseUrl = elmTxtServerUrl.value;
  const apiKey = elmTxtApiKey.value;
  let url = new URL("/api/feeds", baseUrl);

  let json = JSON.stringify({site_url: siteUrl});
  let response;
  try {
    const init = { method: "POST", signal: abortCtrl.signal, headers: {} };
    if (apiKey) init.headers["X-API-KEY"] = apiKey;
    if (json) {
      init.body = json;
      init.headers["Content-Type"] = "application/json";
    }
    response = await fetch(url.toString(), init);
  }
  catch (error) { finished = true; done(error); return; }

  if (response.status >= 300) {
    let respText = "Got status code " + response.status;
    let respData = null;
    try { respData = await response.json(); }
    catch {}
    if (respData != null && respData.hasOwnProperty("error"))
      respText += ": " + respData.error;
    finished = true;
    done(respText);
    return;
  }

  let respData = null;
  if (response.status != 204) {
    try { respData = await response.json(); }
    catch (error) { finished = true; done(error); return; }
  }
  finished = true;
  done(null);
}