import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import { SubmitConnectorPageView } from "./SubmitConnectorPage";

const noop = () => undefined;

function render(
  props: Partial<React.ComponentProps<typeof SubmitConnectorPageView>> = {},
) {
  return renderToStaticMarkup(
    React.createElement(SubmitConnectorPageView, {
      manifestText: "",
      sourceUrl: "",
      description: "",
      screenshots: "",
      status: "idle",
      error: "",
      onManifestTextChange: noop,
      onSourceUrlChange: noop,
      onDescriptionChange: noop,
      onScreenshotsChange: noop,
      onSubmit: noop,
      ...props,
    }),
  );
}

function assertMatch(input: string, pattern: RegExp) {
  if (!pattern.test(input)) {
    throw new Error(`Expected ${pattern} to match ${input}`);
  }
}

const initial = render();
assertMatch(initial, /Submit a community connector/);
assertMatch(initial, /Manifest JSON/);
assertMatch(initial, /Source URL/);
assertMatch(initial, /Review description/);
assertMatch(initial, /Screenshot URLs/);
assertMatch(initial, /Submit connector/);

const submitted = render({ status: "submitted" });
assertMatch(submitted, /Connector submitted for review/);
assertMatch(submitted, /email the developer/);

const error = render({ error: "Manifest must be valid JSON." });
assertMatch(error, /Manifest must be valid JSON/);

console.log("SubmitConnectorPage states render correctly");
