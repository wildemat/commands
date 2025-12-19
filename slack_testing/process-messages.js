#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const INPUT_FILE = path.join(
  __dirname,
  "agent-builder-messages-formatted.ndjson"
);
const OUTPUT_FILE = path.join(__dirname, "agent-builder-processed.ndjson");

/**
 * Extract only the required fields from a message
 */
function extractFields(msg, parentThreadId = null) {
  const result = {
    client_msg_id: msg.client_msg_id || null,
    user: msg.user || null,
    text: msg.text || null,
    ts: msg.ts || null,
  };

  // Extract file permalinks if files exist
  if (msg.files && Array.isArray(msg.files)) {
    const permalinks = msg.files.map((f) => f.permalink).filter((p) => p); // filter out empty/null permalinks
    if (permalinks.length > 0) {
      result.files_permalink = permalinks;
    }
  }

  // Add parent_thread_id if this is a thread reply
  if (parentThreadId !== null) {
    result.parent_thread_id = parentThreadId;
  }

  return result;
}

/**
 * Process a message and its thread replies
 */
function processMessage(msg) {
  const results = [];

  // Extract fields from the main message
  const mainMessage = extractFields(msg);
  results.push(mainMessage);

  // Process thread replies if they exist
  if (
    msg.slackdump_thread_replies &&
    Array.isArray(msg.slackdump_thread_replies)
  ) {
    const parentId = msg.client_msg_id;

    for (const reply of msg.slackdump_thread_replies) {
      // Skip the first reply if it's the same as the parent (often the case in Slack)
      if (reply.client_msg_id === parentId && reply.ts === msg.ts) {
        continue;
      }

      const replyMessage = extractFields(reply, parentId);
      results.push(replyMessage);
    }
  }

  return results;
}

async function main() {
  console.log(`Reading from: ${INPUT_FILE}`);

  // Read the entire file
  const content = fs.readFileSync(INPUT_FILE, "utf-8");

  // The file contains pretty-printed JSON objects separated by commas
  // Wrap in array brackets to parse as JSON array
  let messages;
  try {
    messages = JSON.parse("[" + content + "]");
  } catch (err) {
    console.error("Failed to parse JSON:", err.message);
    process.exit(1);
  }

  console.log(`Parsed ${messages.length} messages`);

  // Process all messages
  const outputLines = [];
  let processedCount = 0;
  let threadReplyCount = 0;

  for (const msg of messages) {
    const processed = processMessage(msg);
    for (const item of processed) {
      outputLines.push(JSON.stringify(item));
      if (item.parent_thread_id) {
        threadReplyCount++;
      }
    }
    processedCount++;
  }

  // Write output as NDJSON (one JSON object per line)
  fs.writeFileSync(OUTPUT_FILE, outputLines.join("\n") + "\n", "utf-8");

  console.log(`Processed ${processedCount} parent messages`);
  console.log(`Extracted ${threadReplyCount} thread replies`);
  console.log(`Total output records: ${outputLines.length}`);
  console.log(`Output written to: ${OUTPUT_FILE}`);
}

main().catch((err) => {
  console.error("Error:", err);
  process.exit(1);
});
