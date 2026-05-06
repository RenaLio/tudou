# PRD: Add Kimi Coding Platform Provider

## Background
- Add a new provider platform for Moonshot Kimi coding gateway.
- This platform should support both OpenAI-compatible and Anthropic-compatible request formats.

## Requirements
1. Add a new backend provider platform for Kimi coding.
   - PlatformId: `kimi-for-coding`
2. Endpoint compatibility:
   - OpenAI-compatible base URL: `https://api.kimi.com/coding/v1`
   - Anthropic-compatible base URL: `https://api.kimi.com/coding/`
3. Models handling:
   - Use dynamic model fetch to validate API key availability.
   - Final returned model list should be the **union** of:
     - remotely fetched models
     - docs-defined static model list
4. Static model list includes:
   - `kimi-k2.6`
   - `kimi-for-coding`
   - `kimi-k2.5`
   - `kimi-k2-thinking`

## Scope
- Backend only (provider platform integration + backend platform wiring).
- No frontend changes in this task.

## Acceptance Criteria
1. New provider platform is selectable and routed by backend provider switch.
2. Platform supports both OpenAI ChatCompletions and Anthropic Messages paths.
3. `Models()` implementation merges fetched models with static list and deduplicates.
4. Existing provider behavior remains unaffected.
