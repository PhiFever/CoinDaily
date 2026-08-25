# Feature Specification: 多市场小时行情与资金动向报表

**Created**: 2026-08-25

**Status**: Ready for implementation after context compaction

**Input**: 在保留 CoinGecko 加密货币行情的基础上，以尽量小的改动加入 QQQ、SPCX 美股标的行情和 Hyperliquid `xyz:ZHIPU` 永续行情，并通过现有邮件与 Discord 通知发送一份分区报表。

## Problem Statement

CoinDaily 当前只通过 CoinGecko 获取加密货币行情，并将所有数据建模和展示为“币种”。实际部署使用 cron 每小时以单次运行模式启动，因此用户希望同一份小时报告也能覆盖关注的美股/ETF 标的和 Hyperliquid 永续合约。

QQQ、SPCX 所需的是美股标的本身的价格，而不是同名或类似的链上代币价格。Hyperliquid 上的 `xyz:ZHIPU` 则需要保留永续合约特有的价格和持仓语义。三类资产不能被强行解释成同一种市场数据：股票没有加密货币市值字段，永续合约的持仓量和资金费率也不能被当成股票或币种市值。

用户还希望观察“资金流入/流出量之类”的信号。但现有数据源并不直接提供跨市场、定义一致的真实净资金流数据。报告必须提供可验证的成交、价格、持仓与资金费率代理指标，并明确称为“资金动向”，避免把成交额或价格上涨误报为真实净流入。

当前调度流程还会在 CoinGecko 获取失败时提前终止。引入多个行情源后，任何单一来源的短暂故障都不应阻止其他有效行情形成报告。

## Solution

CoinDaily 每次运行将分别收集三类行情，并通过现有邮件和 Discord 渠道发送一份“市场行情报表”：

- 加密货币区块继续使用 CoinGecko，并保留现有币种配置。
- 美股/ETF 区块使用 Alpaca 的 15 分钟延迟 SIP 数据，初始跟踪 QQQ 和 SPCX。
- 永续合约区块使用 Hyperliquid 公开 Info API，初始跟踪 `xyz:ZHIPU`。

报告按资产类别分区，各区块只展示语义正确的指标。QQQ、SPCX 展示延迟行情、日内变化、最近完整小时的成交额与 VWAP；`xyz:ZHIPU` 以标记价格为主，并展示预言机价格、溢折价、24 小时变化、每小时资金费率、名义持仓量、较上一份有效快照的持仓变化和最近完整小时的估算成交额；加密货币展示现有价格指标，并补充 24 小时市值变化作为资金动向代理。

各数据源独立失败。只要至少一个区块取得有效数据，系统就发送部分报告，并在失败区块明确标注错误。只有所有已配置数据源都无法提供数据时，系统才不发送空报告。

为了计算 `xyz:ZHIPU` 较上一小时的名义持仓变化，单次运行会在配置文件所在目录维护一个原子更新的本地状态文件。该文件只保存计算下一次差值所需的非敏感行情数值和时间戳。

## User Stories

1. As a CoinDaily user, I want one report to contain crypto, US stock/ETF, and perpetual-market data, so that I can monitor my selected markets without operating separate programs.
2. As a CoinDaily user, I want QQQ to represent the listed Invesco QQQ ETF, so that a similarly named token or index is not silently substituted for my intended asset.
3. As a CoinDaily user, I want SPCX to represent the listed US equity, so that the report reflects the underlying security rather than an issuer-specific tokenized representation.
4. As a CoinDaily user, I want QQQ and SPCX to come from one stock-market feed, so that their prices use a consistent market-data definition.
5. As a CoinDaily user, I want Alpaca's delayed SIP feed to be used for stocks, so that hourly monitoring receives broad US market coverage without requiring a paid real-time SIP subscription.
6. As a CoinDaily user, I want the stock section to state that its data is delayed, so that I do not mistake it for an executable real-time quote.
7. As a CoinDaily user, I want stock prices to retain their latest valid value outside market hours, so that the report remains informative when the market is closed.
8. As a CoinDaily user, I want stock quote timestamps to be visible, so that a repeated close or delayed quote is not mistaken for a fresh trade.
9. As a CoinDaily user, I want to configure a list of stock symbols, so that additional supported US stocks or ETFs can be added without code changes.
10. As a CoinDaily user, I want Alpaca credentials to remain optional when no stocks are configured, so that existing crypto-only installations remain compatible.
11. As a CoinDaily user, I want startup to reject a stock configuration with missing Alpaca credentials, so that a configuration mistake cannot silently suppress the stock section.
12. As a CoinDaily user, I want Alpaca credentials to stay in my local configuration, so that no secret is embedded in source code, examples, logs, state files, or reports.
13. As a CoinDaily user, I want to monitor the exact Hyperliquid contract `xyz:ZHIPU`, so that the report follows the intended HIP-3 perpetual market.
14. As a CoinDaily user, I want the ZHIPU mark price to be the primary displayed price, so that the reported value matches the contract's mark-price semantics.
15. As a CoinDaily user, I want the ZHIPU oracle price shown beside the mark price, so that I can see the external reference used by the perpetual market.
16. As a CoinDaily user, I want the mark-to-oracle spread shown as both direction and percentage, so that premium or discount is immediately visible.
17. As a CoinDaily user, I want the ZHIPU 24-hour change calculated from mark price and the prior-day reference price, so that the change uses a documented, reproducible basis.
18. As a CoinDaily user, I want the current hourly funding rate displayed, so that the cost and directional pressure of holding the perpetual are visible.
19. As a CoinDaily user, I want ZHIPU open interest expressed as a USD notional estimate, so that the number is comparable over time and not presented as an unexplained base-unit quantity.
20. As a CoinDaily user, I want the change in ZHIPU notional open interest since the previous valid hourly run, so that I can distinguish position expansion from position contraction.
21. As a CoinDaily user, I want first-run or stale-snapshot reports to show that open-interest comparison is unavailable, so that missing history is never rendered as zero change.
22. As a CoinDaily user, I want a recent hourly ZHIPU turnover estimate, so that I can compare current activity with the position and funding signals.
23. As a CoinDaily user, I want these activity indicators labeled “资金动向”, so that they are not misrepresented as audited net capital inflow or outflow.
24. As a CoinDaily user, I want stock funding activity represented by hourly turnover, VWAP, and price direction, so that the report uses data the selected provider actually supplies.
25. As a CoinDaily user, I want crypto funding activity represented by 24-hour volume and market-cap change, so that the report provides a useful proxy without inventing buy/sell flow data.
26. As a CoinDaily user, I want metrics that are not meaningful for an asset type to be omitted or shown as unavailable, so that a zero value is not confused with a real observation.
27. As a CoinDaily user, I want one notification with three clearly labeled sections, so that hourly cron runs do not generate three separate messages.
28. As a CoinDaily user, I want the report title and wording to describe a general market report, so that stock and perpetual data are not labeled as cryptocurrency.
29. As a CoinDaily user, I want every section to identify its data source and price timestamp, so that differences in freshness and market semantics are transparent.
30. As a CoinDaily user, I want email and Discord to contain the same core observations, so that the chosen notification channel does not change the meaning of the report.
31. As a CoinDaily user, I want a single data-source failure to produce a partial report with an explicit warning, so that healthy sources remain useful.
32. As a CoinDaily user, I want all configured sources to fail before the program suppresses a report, so that empty notifications are avoided without discarding valid data.
33. As a CoinDaily operator, I want each provider failure logged with its provider name and useful context, so that I can diagnose a missing section without exposing credentials.
34. As a CoinDaily operator, I want notification-channel failures to remain independent, so that an email failure does not prevent Discord delivery and vice versa.
35. As a CoinDaily operator, I want the existing proxy configuration reused by the new HTTP clients, so that network access behaves consistently with CoinGecko and Discord.
36. As a CoinDaily operator, I want the existing `-once` execution path to collect all configured markets, so that the installed hourly cron entry requires no change.
37. As a CoinDaily operator, I want daemon scheduling behavior to remain available, so that installations not using cron retain their current operating mode.
38. As a CoinDaily operator, I want the previous ZHIPU snapshot saved atomically, so that interruption during a write cannot leave a partially written state file.
39. As a CoinDaily operator, I want a missing or corrupt state file to degrade gracefully, so that reporting continues and a fresh comparison baseline is established.
40. As a CoinDaily operator, I want backwards-compatible configuration defaults, so that upgrading the binary does not require enabling Alpaca or Hyperliquid.
41. As a CoinDaily operator, I want sample configuration and documentation for each provider, so that I can enable the feature without reading source code.
42. As a CoinDaily operator, I want no live external APIs used by automated tests, so that the test suite stays deterministic and does not consume quotas.

## Implementation Decisions

- The application remains a single Go executable. The current cron entry, which invokes single-run mode at the top of every hour, is not modified by this feature.
- Market data is represented as a report containing typed sections rather than coercing stocks and perpetuals into the existing cryptocurrency price type. Shared presentation fields may be normalized, while asset-specific fields remain explicit and optional.
- CoinGecko remains the cryptocurrency source and continues to use the existing markets endpoint and retry/proxy behavior. Its response model is extended to retain 24-hour market-cap change fields already supplied by that endpoint.
- Alpaca is the sole source for QQQ and SPCX. The initial feed is `delayed_sip`; the report must visibly identify the 15-minute delayed SIP semantics.
- The stock client uses authenticated REST requests and batches configured symbols where the API supports batching. It obtains the latest delayed stock observation, previous regular-session close, and recent completed hourly bar data needed for price change, VWAP, volume, and approximate dollar turnover.
- The stock primary price is the latest valid delayed trade or equivalent snapshot observation. Daily percentage change compares that value with the previous regular-session close. The observation timestamp is retained and displayed.
- Stock hourly turnover is derived from the provider's completed hourly volume and VWAP. It is an activity measure, not a net-flow measure.
- Hyperliquid uses the unauthenticated public Info API. The configured symbol includes its HIP-3 DEX prefix; the initial value is `xyz:ZHIPU`.
- Hyperliquid metadata and asset contexts are matched by their aligned universe index. The implementation must not assume that `ZHIPU` has a stable numeric asset index.
- ZHIPU's primary price is `markPx`; `oraclePx` is secondary. Absolute spread is mark minus oracle, and percentage spread is that difference divided by oracle.
- ZHIPU 24-hour price change is calculated from `markPx` relative to `prevDayPx`. Missing or zero reference data yields an unavailable value instead of division by zero.
- ZHIPU funding is displayed as the current hourly funding rate in percentage form and is not annualized.
- ZHIPU current notional open interest is estimated as base-unit open interest multiplied by mark price. The report labels the value as notional open interest, not capital deposited.
- ZHIPU open-interest change is current notional open interest minus the prior valid snapshot. Positive change is labeled 增仓, negative change is labeled 减仓, and neither direction is described as net money flow.
- A prior open-interest snapshot is valid for hourly comparison only when it is no more than 90 minutes old. Older, missing, malformed, non-finite, or symbol-mismatched snapshots do not produce a delta.
- Hyperliquid hourly candles are requested for the configured contract. Because the candle volume is a base-asset quantity, recent hourly dollar turnover is an estimate derived from volume and a representative candle price and is labeled accordingly.
- “资金动向” is the umbrella presentation label. The application does not manufacture a single cross-market inflow number or rank unlike metrics as if they were directly comparable.
- The local state file is named `.coindaily-state.json` and is located beside the active configuration file rather than relative to the cron working directory. It contains only versioned per-contract observation timestamps and prior notional open interest values.
- State updates use a temporary file followed by an atomic rename. A state read or write failure is logged and only disables the corresponding comparison; it does not suppress otherwise valid market data.
- The state baseline is advanced after a successful Hyperliquid observation has been processed, regardless of notification-channel success. This keeps the next comparison interval tied to observations rather than delivery outages.
- Configuration gains an optional Alpaca section containing API key, secret key, and feed selection, plus a stock-symbol list. When the stock list is empty, Alpaca credentials are not required and no stock request occurs. When it is non-empty, both credentials are required.
- Configuration gains an optional Hyperliquid section containing a perpetual-symbol list. No trading key, wallet, address, or authentication material is accepted or required.
- Existing configuration remains valid without the new sections. The initial example configuration documents QQQ, SPCX, and `xyz:ZHIPU` but uses placeholders only and never modifies or commits real credentials.
- New provider clients follow the existing standard-library HTTP style, including bounded timeouts, optional proxy support, status validation, response parsing, and contextual errors. No provider SDK dependency is required.
- Data sources are collected independently. A source error becomes section-level report status rather than causing immediate termination of the entire report run.
- If at least one configured section has valid observations, email and Discord delivery proceed with both successful sections and warnings for failed configured sections. If all configured sources fail or return no usable observations, no notification is sent and the aggregate failure is logged.
- Email and Discord remain independent delivery channels. Failure in one channel never prevents attempting the other.
- The report title changes from a cryptocurrency-only daily title to a general market report title and includes the generation timestamp in the configured host timezone.
- Email renders three semantic sections with section-specific columns. Discord renders the same core values in compact fields while preserving existing platform length safeguards. Unavailable values render as `N/A`, never as a fabricated zero.
- Provider name, feed type, and observation freshness are included in each section footer or heading. Credentials, raw authorization errors, and local state contents are never rendered in notifications.
- The report continues to support the existing configured coins and notification channels without requiring the user to change the hourly cron schedule.

## Testing Decisions

- Tests assert externally observable behavior: accepted/rejected configuration, outbound provider requests, parsed market observations, state transitions, report content, partial-failure behavior, and notification attempts. They do not assert private helper call order or concrete internal struct layout.
- The primary test seam is the single-report execution operation. It receives controllable market collectors, state storage, clock, and notification boundaries so one test can exercise collection, report assembly, state comparison, partial failure, and delivery at the highest practical level.
- Existing scheduler initialization and report-generation tests are the prior art. They are extended around the primary execution seam instead of adding tests for every internal helper.
- Configuration tests cover legacy configurations, empty optional lists, valid Alpaca stock configuration, incomplete credentials, Hyperliquid-only additions, invalid feed values, and symbols containing HIP-3 DEX prefixes.
- HTTP client contract tests use local test servers and configurable base URLs. They verify request method, path, headers, query/body parameters, proxy-independent parsing, response mapping, non-success status handling, malformed JSON, missing symbols, and numeric strings.
- Alpaca contract tests cover batched QQQ/SPCX lookup, delayed SIP selection, latest observation timestamps, previous-close change, completed hourly VWAP/volume, closed-market responses, and a missing symbol in an otherwise successful response.
- Hyperliquid contract tests cover discovery by universe name, index alignment between metadata and contexts, mark/oracle spread, prior-day change, funding conversion, notional open interest, candle turnover estimate, absent contract, null fields, and malformed numeric strings.
- State-store tests use temporary directories. They cover first run, valid prior snapshot, a snapshot older than 90 minutes, symbol mismatch, corrupt JSON, atomic replacement, and write failure without report suppression.
- Report tests cover all three populated sections, each individual section alone, source warnings, `N/A` rendering, positive and negative values, timestamps, delayed-feed labeling, mark/oracle premium and discount, and the absence of “净流入” claims.
- Partial-failure orchestration tests cover each source failing alone, two sources failing, all sources failing, empty successful responses, and independent email/Discord delivery failures.
- Discord tests retain character-limit coverage and add coverage that warning text and multi-section fields are truncated without changing numeric meaning or producing an invalid embed.
- No automated test calls CoinGecko, Alpaca, Hyperliquid, SMTP, or Discord over the public internet. Live credentialed verification is a separate manual step after local configuration.
- The repository-wide Go test suite must pass before implementation is considered complete.

## Out of Scope

- Trading, order submission, leverage management, wallet integration, deposits, withdrawals, or custody on Alpaca or Hyperliquid.
- Collection or storage of any Hyperliquid private key, API wallet, or user account data.
- Replacing QQQ with Hyperliquid `xyz:XYZ100`; these are different instruments and must not be presented as equivalents.
- Tracking tokenized-stock issuer instruments for QQQ or SPCX through CoinGecko.
- Claiming true net capital inflow/outflow from OHLCV, turnover, market-cap change, funding, or open-interest data.
- ETF creation/redemption flow, institutional flow, exchange inflow/outflow, wallet flow, or broker order-flow analytics from an additional paid specialist provider.
- Continuous WebSocket ingestion, tick-by-tick aggressor classification, order-book imbalance, cumulative volume delta, or running a Hyperliquid node.
- Purchasing or automatically upgrading an Alpaca market-data subscription. Real-time full-market SIP is not required for the agreed hourly report.
- Changing the system crontab, changing the hourly invocation cadence, or replacing the existing daemon scheduler.
- Building a web dashboard, historical database, charting interface, alert threshold engine, or portfolio profit-and-loss tracker.
- Automatically discovering arbitrary stock or perpetual symbols. Invalid or delisted configured instruments are reported as source/asset errors.
- Publishing this specification to an external issue tracker in this archival step; the user explicitly requested a repository file and commit before manual context compaction.

## Further Notes

- At specification time, Hyperliquid's live metadata lists `xyz:ZHIPU` on the `xyz` HIP-3 DEX and its public asset context supplies mark price, oracle price, prior-day price, funding, open interest, and 24-hour notional volume. Availability is external state and must still be validated on every run.
- Hyperliquid HIP-3 markets are builder-deployed perpetuals. Their oracle and market operation are the deployer's responsibility, so the report must identify the value as a perpetual-market observation rather than an official underlying equity quote.
- Alpaca authentication uses an API key and secret in request headers. The user will create and place free-plan credentials in the local configuration after implementation; credentials must not be pasted into documentation or committed.
- The agreed Alpaca delayed SIP choice prioritizes consistent broad-market coverage over sub-second freshness. A 15-minute delay is acceptable for a cron job that reports once per hour.
- The installed cron entry already invokes single-run mode hourly. Internal `schedule` configuration remains relevant only when the long-running daemon mode is used.
- Implementation begins only after the user manually compacts context and asks the agent to proceed.
