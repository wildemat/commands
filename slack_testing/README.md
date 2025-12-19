getting google auth working: https://github.com/rusq/slackdump/issues/524
https://github.com/rusq/slackdump/blob/master/doc/login-manual.rst#cookie

hackathon channel: https://app.slack.com/client/T0CUZ52US/C09MP284CP6

agent builder channel: https://app.slack.com/client/T0CUZ52US/C08LX7YSHU2

This will prompt to open brave and then just work:

slackdump dump -files=false -workspace=elastic -time-from=2025-10-01T00:00:00 C08LX7YSHU2

it may flag security team for incident response

copied just the message objects into agent-builder-messages-formatted.ndjson

need to postprocess

for each message keep:
client_msg_id,user,text,ts,files.permalink

for messages within a slackdump_thread_replies key:
break them out into the root level and attach a field parent_thread_id using the client_msg_id from the parent object.

upload the processed messages to elastic index

create a new agent tool (agent_builder_slack_messages):

Description:
The agent-builder-messages index contains message threads and their text content from the #agent-builder slack channel.
Search through the text content in the index to discover patterns.

Custom instructions:
If a document contains a defined "parent_thread_id" value, then it should be considered as a child of the document with the matching "client_msg_id" to that "parent_thread_id". Children of a parent thread should all be compiled together and sorted by "ts" in order to gain the full context of a specific message thread.

Initial searches through the index to find patterns should first only search through messages that do not have a defined "parent_thread_id" in order to first find the threads that may match the context the user is asking about. Then the children of those parent threads can be searched through to gain additional context
