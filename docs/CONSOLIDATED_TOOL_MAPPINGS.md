# MCP Tool Parameter Quick Reference

**9 Consolidated Tools** | 41 → 9 (78% reduction) | Better client compatibility

## Core Pattern
```json
{
  "operation": "operation_name",
  "scope": "single|bulk|cross_repo|system|session|workflow|global",
  "options": { /* operation-specific parameters */ }
}
```

## memory_create
- **store_chunk**: `content*(str), session_id*(str), repository(str), tags(arr), files_modified(arr), tools_used(arr)`
- **store_decision**: `decision*(str), rationale*(str), session_id*(str), context(str), repository(str)`
- **create_thread**: `chunk_ids*(arr), name*(str), description(str), repository(str), tags(arr)` ⚠️ Note: chunk_ids must reference existing valid chunks
- **create_alias**: `name*(str), type*(str), target*(str), repository(str), description(str)` ⚠️ Note: simplified parameters
- **create_relationship**: `source_chunk_id*(str), target_chunk_id*(str), relation_type*(str), strength(num), repository(str), description(str)` ⚠️ Note: use 'strength' not 'confidence'
- **auto_detect_relationships**: `chunk_id*(str), session_id*(str), min_confidence(num), enabled_detectors(arr), auto_store(bool)`
- **import_context**: `data*(str), repository*(str), session_id*(str), source(str), metadata(obj), chunking_strategy(str)`
- **bulk_import**: `data*(str), format(str), repository(str), default_session_id(str), default_tags(arr), chunking_strategy(str), conflict_policy(str), validate_chunks(bool), metadata(obj)`

## memory_read
- **search**: `query*(str), repository(str), recency(str), types(arr), limit(int), min_relevance(num)`
- **get_context**: `repository*(str), recent_days(int)`
- **find_similar**: `problem*(str), repository(str), limit(int)`
- **get_patterns**: `repository*(str), timeframe(str)`
- **get_relationships**: `chunk_id*(str), relation_types(arr), direction(str), min_confidence(num), max_depth(int), include_chunks(bool), limit(int)`
- **traverse_graph**: `start_chunk_id*(str), max_depth(int), relation_types(arr), min_confidence(num)`
- **get_threads**: `repository(str), status(str), thread_type(str), session_id(str), include_summary(bool)`
- **search_explained**: `query*(str), repository(str), explain_depth(str), include_relationships(bool), include_citations(bool), min_relevance(num), limit(int)`
- **search_multi_repo**: `query*(str), session_id*(str), repositories(arr), tech_stacks(arr), frameworks(arr), pattern_types(arr), min_confidence(num), max_results(int), include_similar(bool)`
- **resolve_alias**: `alias_name*(str)`
- **list_aliases**: `type(str), repository(str), tags(arr), query(str), sort_by(str), limit(int)`
- **get_bulk_progress**: `operation_id*(str)`

## memory_update
- **update_thread**: `thread_id*(str), status(str), title(str), add_chunks(arr), remove_chunks(arr)`
- **update_relationship**: `relationship_id*(str), confidence(num), user_certainty(num), validation_note(str)`
- **mark_refreshed**: `chunk_id*(str), validation_notes*(str), update_quality_scores(bool)` ⚠️ Note: chunk_id singular
- **resolve_conflicts**: `conflict_ids*(arr), repository(str), strategy_types(arr), max_strategies(int), include_detailed_steps(bool)`
- **bulk_update**: `chunks*(arr), batch_size(int), max_concurrency(int), validate_first(bool), continue_on_error(bool), dry_run(bool), conflict_policy(str)`
- **decay_management**: `repository*(str), session_id*(str), action*(str), config(obj), preview_only(bool), intelligent_mode(bool)`

## memory_delete
- **bulk_delete**: `ids*(arr), batch_size(int), max_concurrency(int), validate_first(bool), continue_on_error(bool), dry_run(bool)`

## memory_analyze
- **cross_repo_patterns**: `session_id*(str), repositories(arr), tech_stacks(arr), pattern_types(arr), min_frequency(int)`
- **find_similar_repositories**: `repository*(str), session_id*(str), similarity_threshold(num), limit(int)`
- **cross_repo_insights**: `session_id*(str), include_tech_distribution(bool), include_success_analytics(bool), include_pattern_frequency(bool)`
- **detect_conflicts**: `repository(str), timeframe(str)`
- **health_dashboard**: `repository*(str), session_id*(str), timeframe(str), include_details(bool), include_recommendations(bool)` ⚠️ Note: both required
- **check_freshness**: `repository*(str), chunk_id(str), include_stale_only(bool), technology_filter(str), generate_alerts(bool)`
- **detect_threads**: `repository*(str), auto_create(bool), min_thread_size(int)`

## memory_intelligence
- **suggest_related**: `current_context*(str), session_id*(str), repository(str), max_suggestions(int), include_patterns(bool)` ⚠️ Note: both current_context AND session_id required
- **auto_insights**: `repository*(str), session_id*(str), timeframe(str), insight_types(arr), min_confidence(num)` ⚠️ Note: both repository AND session_id required  
- **pattern_prediction**: `context*(str), repository*(str), session_id*(str), prediction_type(str), confidence_threshold(num)` ⚠️ Note: context, repository AND session_id required

## memory_transfer
- **export_project**: `repository*(str), session_id*(str), format(str), include_vectors(bool), date_range(obj)` ⚠️ Note: both repository AND session_id required
- **bulk_export**: `format(str), compression(str), include_vectors(bool), include_metadata(bool), include_relations(bool), pretty_print(bool), filter(obj), sorting(obj), pagination(obj)`
- **continuity**: `repository*(str), session_id(str), include_suggestions(bool)`
- **import_context**: `data*(str), repository*(str), session_id*(str), source(str), metadata(obj), chunking_strategy(str)` ⚠️ Note: data, repository AND session_id required

## memory_system
- **health**: No parameters required
- **status**: `repository*(str)`
- **generate_citations**: `query*(str), chunk_ids*(arr), citation_style(str), group_sources(bool), include_context(bool)`
- **create_inline_citation**: `text*(str), response_id*(str), format(str)`
- **get_documentation**: `doc_type(str="both")` - Stream tool documentation. Options: "mappings", "examples", "both"

## memory_tasks
- **todo_write**: `todos*(arr), session_id(str="default"), repository(str="unknown")`
- **todo_read**: `session_id(str="default")`
- **todo_update**: `tool_name*(str), session_id(str="default"), tool_context(obj)`
- **session_create**: `session_id*(str), repository(str="unknown")`
- **session_end**: `session_id*(str), outcome(str="success")`
- **session_list**: No parameters required
- **workflow_analyze**: `session_id*(str)`
- **task_completion_stats**: No parameters required

## Legend
- `*` = Required parameter
- `(type)` = Parameter type: str=string, int=integer, num=number, bool=boolean, arr=array, obj=object
- `(default)` = Default value if not provided
- ⚠️ = Common error - pay attention to these parameters

## Quick Examples
```json
// Store a bug fix
{"operation": "store_chunk", "scope": "single", "options": {"content": "Fixed auth bug", "session_id": "fix-123"}}

// Search for similar issues  
{"operation": "search", "scope": "single", "options": {"query": "authentication bug", "limit": 5}}

// Get system health
{"operation": "health", "scope": "system", "options": {}}

// Create todo list
{"operation": "todo_write", "scope": "session", "options": {"todos": [{"id": "1", "content": "Fix bug", "status": "pending", "priority": "high"}]}}
```

**Memory Usage**: ~800 tokens (vs ~2000 tokens original)