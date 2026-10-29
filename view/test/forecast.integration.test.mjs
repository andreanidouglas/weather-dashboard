import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test } from "node:test";
import { Window } from "happy-dom";

const forecastFixture = {
  days: Array.from({ length: 7 }, (_, index) => ({
    date: `Day ${index + 1}`,
    min_temp: 18,
    max_temp: 22,
    condition: "Clear",
    description: index === 0 ? "<img src=x onerror=alert(1)>" : "clear sky",
    humidity: 51,
    precipitation: 0.2,
  })),
};

async function waitFor(assertion, message) {
  const deadline = Date.now() + 1500;
  while (Date.now() < deadline) {
    if (assertion()) return;
    await new Promise((resolve) => setTimeout(resolve, 5));
  }
  assert.fail(message);
}

test("location detail button opens and updates the seven-day forecast modal", async () => {
  const window = new Window({ url: "http://weather.test/" });
  const globals = [
    "window",
    "document",
    "MutationObserver",
    "CustomEvent",
    "Element",
    "HTMLElement",
    "Node",
    "Document",
    "ShadowRoot",
    "localStorage",
    "requestAnimationFrame",
    "cancelAnimationFrame",
    "getComputedStyle",
  ];
  const originalGlobals = new Map(
    globals.map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)]),
  );
  for (const key of globals) globalThis[key] = window[key];

  const requests = [];
  let failRequest = false;
  const mockFetch = async (input) => {
    requests.push(new URL(input, window.location.href));
    if (failRequest) return { ok: false, status: 502 };
    return { ok: true, json: async () => forecastFixture };
  };
  window.fetch = mockFetch;
  globalThis.fetch = mockFetch;

  let alpine;
  try {
    const pagePath = new URL("../src/index.html", import.meta.url);
    const page = await readFile(pagePath, "utf8");
    const pageWithoutScripts = page.replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, "");
    window.document.write(pageWithoutScripts);
    window.document.close();
    window.localStorage.setItem("x_cities", "[]");

    const grid = window.document.querySelector("#grid");
    grid.insertAdjacentHTML(
      "beforeend",
      `<div class="weather-fixture">
        <button id="fahrenheit-detail" type="button" data-lat="41.9" data-lon="12.5"
          data-city="Rome, IT" data-fahrenheit="true"
          x-on:click="$dispatch('forecast-open', {lat: Number($el.dataset.lat), lon: Number($el.dataset.lon), city: $el.dataset.city, fahrenheit: $el.dataset.fahrenheit === 'true'})">7-day forecast</button>
        <button id="metric-detail" type="button" data-lat="48.85" data-lon="2.35"
          data-city="Paris, FR" data-fahrenheit="false"
          x-on:click="$dispatch('forecast-open', {lat: Number($el.dataset.lat), lon: Number($el.dataset.lon), city: $el.dataset.city, fahrenheit: $el.dataset.fahrenheit === 'true'})">7-day forecast</button>
      </div>`,
    );

    ({ Alpine: alpine } = await import("alpinejs"));
    window.Alpine = alpine;
    alpine.start();

    const app = window.document.querySelector('div[x-data*="forecastOpen"]');
    await waitFor(() => app?._x_dataStack, "Alpine did not initialize the dashboard");

    const dialog = window.document.querySelector('[role="dialog"]');
    const fahrenheitButton = window.document.querySelector("#fahrenheit-detail");
    fahrenheitButton.click();

    await waitFor(
      () => dialog.querySelectorAll("article").length === 7,
      "forecast response did not render seven daily cards",
    );
    assert.notEqual(dialog.style.display, "none", "dialog should be visible after opening");
    assert.equal(dialog.querySelector("#forecast-title").textContent, "Rome, IT");
    assert.equal(requests.length, 1);
    assert.equal(requests[0].pathname, "/api/forecast");
    assert.equal(requests[0].searchParams.get("lat"), "41.9");
    assert.equal(requests[0].searchParams.get("lon"), "12.5");
    assert.equal(requests[0].searchParams.get("fahrenheit"), "yes");
    assert.match(dialog.textContent, /22°F/);
    assert.match(dialog.textContent, /18°F/);
    assert.match(dialog.textContent, /Humidity 51% · Rain 20%/);
    assert.match(dialog.textContent, /<img src=x onerror=alert\(1\)>/);
    assert.equal(dialog.querySelector("img"), null, "upstream text must not become executable markup");

    dialog.querySelector('[aria-label="Close forecast"]').click();
    await waitFor(() => dialog.style.display === "none", "close button did not hide the dialog");

    window.document.querySelector("#metric-detail").click();
    await waitFor(() => requests.length === 2, "metric forecast request was not made");
    await waitFor(() => dialog.querySelectorAll("article").length === 7, "metric forecast did not render");
    assert.equal(dialog.querySelector("#forecast-title").textContent, "Paris, FR");
    assert.equal(requests[1].searchParams.get("fahrenheit"), null);
    assert.match(dialog.textContent, /22°C/);

    failRequest = true;
    fahrenheitButton.click();
    await waitFor(
      () => dialog.querySelector('[role="alert"]')?.textContent.includes("Unable to load"),
      "failed forecast request did not show an error",
    );
    window.dispatchEvent(new window.KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
    await waitFor(() => dialog.style.display === "none", "Escape did not close the dialog");
  } finally {
    alpine?.stopObservingMutations();
    window.happyDOM.abort();
    window.close();
    for (const [key, descriptor] of originalGlobals) {
      if (descriptor === undefined) delete globalThis[key];
      else Object.defineProperty(globalThis, key, descriptor);
    }
  }
});
