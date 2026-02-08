package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	activitytemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/activity_template"
	stagetemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/stage_template"
	workflowtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/workflow/workflow_template"
)

// DefaultCacheTTL is the default time-to-live for cached templates
const DefaultCacheTTL = 5 * time.Minute

// cacheEntry wraps a cached item with its expiration time
type cacheEntry[T any] struct {
	data      T
	expiresAt time.Time
}

// isExpired checks if the cache entry has expired
func (e *cacheEntry[T]) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// TemplateCache provides thread-safe caching for workflow, stage, and activity templates.
// It uses a hybrid approach: preload known templates at startup, lazy load on cache miss.
type TemplateCache struct {
	repos EngineRepositories
	ttl   time.Duration

	mu                 sync.RWMutex
	workflowTemplates  map[string]*cacheEntry[*workflowtemplatepb.WorkflowTemplate]
	stageTemplates     map[string]*cacheEntry[[]*stagetemplatepb.StageTemplate] // keyed by workflow_template_id
	stageTemplatesById map[string]*cacheEntry[*stagetemplatepb.StageTemplate]   // keyed by stage_template_id
	activityTemplates  map[string]*cacheEntry[*activitytemplatepb.ActivityTemplate]
}

// NewTemplateCache creates a new template cache with the default TTL
func NewTemplateCache(repos EngineRepositories) *TemplateCache {
	return NewTemplateCacheWithTTL(repos, DefaultCacheTTL)
}

// NewTemplateCacheWithTTL creates a new template cache with a custom TTL
func NewTemplateCacheWithTTL(repos EngineRepositories, ttl time.Duration) *TemplateCache {
	return &TemplateCache{
		repos:              repos,
		ttl:                ttl,
		workflowTemplates:  make(map[string]*cacheEntry[*workflowtemplatepb.WorkflowTemplate]),
		stageTemplates:     make(map[string]*cacheEntry[[]*stagetemplatepb.StageTemplate]),
		stageTemplatesById: make(map[string]*cacheEntry[*stagetemplatepb.StageTemplate]),
		activityTemplates:  make(map[string]*cacheEntry[*activitytemplatepb.ActivityTemplate]),
	}
}

// Preload eagerly loads known workflow templates and their associated stage/activity templates.
// This should be called at startup for frequently used workflows.
func (c *TemplateCache) Preload(ctx context.Context, workflowTemplateIDs []string) error {
	start := time.Now()
	log.Printf("[TemplateCache] Preloading %d workflow templates...", len(workflowTemplateIDs))

	var preloadErrors []error

	for _, wtID := range workflowTemplateIDs {
		// Load workflow template
		_, err := c.GetWorkflowTemplate(ctx, wtID)
		if err != nil {
			preloadErrors = append(preloadErrors, fmt.Errorf("workflow template %s: %w", wtID, err))
			continue
		}

		// Load stage templates for this workflow
		stageTemplates, err := c.GetStageTemplates(ctx, wtID)
		if err != nil {
			preloadErrors = append(preloadErrors, fmt.Errorf("stage templates for %s: %w", wtID, err))
			continue
		}

		// Load activity templates for each stage
		for _, st := range stageTemplates {
			_, err := c.GetActivityTemplatesForStage(ctx, st.Id)
			if err != nil {
				preloadErrors = append(preloadErrors, fmt.Errorf("activity templates for stage %s: %w", st.Id, err))
			}
		}
	}

	log.Printf("[TemplateCache] Preload completed in %v. Errors: %d", time.Since(start), len(preloadErrors))

	if len(preloadErrors) > 0 {
		return fmt.Errorf("preload encountered %d errors: %v", len(preloadErrors), preloadErrors)
	}

	return nil
}

// GetWorkflowTemplate retrieves a workflow template from cache or fetches from repository on miss.
func (c *TemplateCache) GetWorkflowTemplate(ctx context.Context, id string) (*workflowtemplatepb.WorkflowTemplate, error) {
	// Check cache first (read lock)
	c.mu.RLock()
	if entry, exists := c.workflowTemplates[id]; exists && !entry.isExpired() {
		c.mu.RUnlock()
		return entry.data, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from repository
	start := time.Now()
	res, err := c.repos.WorkflowTemplate.ReadWorkflowTemplate(ctx, &workflowtemplatepb.ReadWorkflowTemplateRequest{
		Data: &workflowtemplatepb.WorkflowTemplate{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow template: %w", err)
	}
	if res == nil || len(res.Data) == 0 {
		return nil, fmt.Errorf("workflow template not found: %s", id)
	}

	template := res.Data[0]
	log.Printf("[TemplateCache] Cache miss for workflow template %s, fetched in %v", id, time.Since(start))

	// Store in cache (write lock)
	c.mu.Lock()
	c.workflowTemplates[id] = &cacheEntry[*workflowtemplatepb.WorkflowTemplate]{
		data:      template,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return template, nil
}

// GetStageTemplates retrieves stage templates for a workflow from cache or fetches from repository.
// Results are sorted by order_index in ascending order.
func (c *TemplateCache) GetStageTemplates(ctx context.Context, workflowTemplateID string) ([]*stagetemplatepb.StageTemplate, error) {
	// Check cache first (read lock)
	c.mu.RLock()
	if entry, exists := c.stageTemplates[workflowTemplateID]; exists && !entry.isExpired() {
		c.mu.RUnlock()
		return entry.data, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from repository
	start := time.Now()
	res, err := c.repos.StageTemplate.ListStageTemplates(ctx, &stagetemplatepb.ListStageTemplatesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "workflow_template_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    workflowTemplateID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list stage templates: %w", err)
	}

	templates := res.Data
	if templates == nil {
		templates = []*stagetemplatepb.StageTemplate{}
	}

	// Sort by order_index in memory (to avoid Firestore composite index requirement)
	sortStageTemplatesByOrderIndex(templates)

	log.Printf("[TemplateCache] Cache miss for stage templates (workflow %s), fetched %d in %v",
		workflowTemplateID, len(templates), time.Since(start))

	// Store in cache (write lock)
	c.mu.Lock()
	c.stageTemplates[workflowTemplateID] = &cacheEntry[[]*stagetemplatepb.StageTemplate]{
		data:      templates,
		expiresAt: time.Now().Add(c.ttl),
	}
	// Also cache each individual stage template by ID
	for _, tmpl := range templates {
		c.stageTemplatesById[tmpl.Id] = &cacheEntry[*stagetemplatepb.StageTemplate]{
			data:      tmpl,
			expiresAt: time.Now().Add(c.ttl),
		}
	}
	c.mu.Unlock()

	return templates, nil
}

// GetStageTemplate retrieves a single stage template from cache or fetches from repository.
func (c *TemplateCache) GetStageTemplate(ctx context.Context, id string) (*stagetemplatepb.StageTemplate, error) {
	// Check cache first (read lock)
	c.mu.RLock()
	if entry, exists := c.stageTemplatesById[id]; exists && !entry.isExpired() {
		c.mu.RUnlock()
		return entry.data, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from repository
	start := time.Now()
	res, err := c.repos.StageTemplate.ReadStageTemplate(ctx, &stagetemplatepb.ReadStageTemplateRequest{
		Data: &stagetemplatepb.StageTemplate{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read stage template: %w", err)
	}
	if res == nil || len(res.Data) == 0 {
		return nil, fmt.Errorf("stage template not found: %s", id)
	}

	template := res.Data[0]
	log.Printf("[TemplateCache] Cache miss for stage template %s, fetched in %v", id, time.Since(start))

	// Store in cache (write lock)
	c.mu.Lock()
	c.stageTemplatesById[id] = &cacheEntry[*stagetemplatepb.StageTemplate]{
		data:      template,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return template, nil
}

// GetActivityTemplate retrieves a single activity template from cache or fetches from repository.
func (c *TemplateCache) GetActivityTemplate(ctx context.Context, id string) (*activitytemplatepb.ActivityTemplate, error) {
	// Check cache first (read lock)
	c.mu.RLock()
	if entry, exists := c.activityTemplates[id]; exists && !entry.isExpired() {
		c.mu.RUnlock()
		return entry.data, nil
	}
	c.mu.RUnlock()

	// Cache miss - fetch from repository
	start := time.Now()
	res, err := c.repos.ActivityTemplate.ReadActivityTemplate(ctx, &activitytemplatepb.ReadActivityTemplateRequest{
		Data: &activitytemplatepb.ActivityTemplate{Id: id},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read activity template: %w", err)
	}
	if res == nil || len(res.Data) == 0 {
		return nil, fmt.Errorf("activity template not found: %s", id)
	}

	template := res.Data[0]
	log.Printf("[TemplateCache] Cache miss for activity template %s, fetched in %v", id, time.Since(start))

	// Store in cache (write lock)
	c.mu.Lock()
	c.activityTemplates[id] = &cacheEntry[*activitytemplatepb.ActivityTemplate]{
		data:      template,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return template, nil
}

// GetActivityTemplatesForStage retrieves all activity templates for a stage template.
// This is a convenience method that fetches and caches individual activity templates.
func (c *TemplateCache) GetActivityTemplatesForStage(ctx context.Context, stageTemplateID string) ([]*activitytemplatepb.ActivityTemplate, error) {
	start := time.Now()

	res, err := c.repos.ActivityTemplate.ListActivityTemplates(ctx, &activitytemplatepb.ListActivityTemplatesRequest{
		Filters: &commonpb.FilterRequest{
			Filters: []*commonpb.TypedFilter{
				{
					Field: "stage_template_id",
					FilterType: &commonpb.TypedFilter_StringFilter{
						StringFilter: &commonpb.StringFilter{
							Value:    stageTemplateID,
							Operator: commonpb.StringOperator_STRING_EQUALS,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activity templates: %w", err)
	}

	templates := res.Data
	if templates == nil {
		templates = []*activitytemplatepb.ActivityTemplate{}
	}

	// Cache each individual activity template for later lookup by ID
	c.mu.Lock()
	for _, tmpl := range templates {
		c.activityTemplates[tmpl.Id] = &cacheEntry[*activitytemplatepb.ActivityTemplate]{
			data:      tmpl,
			expiresAt: time.Now().Add(c.ttl),
		}
	}
	c.mu.Unlock()

	log.Printf("[TemplateCache] Fetched and cached %d activity templates for stage %s in %v",
		len(templates), stageTemplateID, time.Since(start))

	return templates, nil
}

// Invalidate removes a specific template from the cache by ID.
// This invalidates across all template types.
func (c *TemplateCache) Invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.workflowTemplates, id)
	delete(c.stageTemplates, id)
	delete(c.activityTemplates, id)

	log.Printf("[TemplateCache] Invalidated cache entry: %s", id)
}

// InvalidateWorkflowTemplate removes a workflow template and its associated stage templates from cache.
func (c *TemplateCache) InvalidateWorkflowTemplate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.workflowTemplates, id)
	delete(c.stageTemplates, id) // Stage templates are keyed by workflow_template_id

	log.Printf("[TemplateCache] Invalidated workflow template and associated stages: %s", id)
}

// InvalidateAll clears the entire cache.
func (c *TemplateCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workflowTemplates = make(map[string]*cacheEntry[*workflowtemplatepb.WorkflowTemplate])
	c.stageTemplates = make(map[string]*cacheEntry[[]*stagetemplatepb.StageTemplate])
	c.stageTemplatesById = make(map[string]*cacheEntry[*stagetemplatepb.StageTemplate])
	c.activityTemplates = make(map[string]*cacheEntry[*activitytemplatepb.ActivityTemplate])

	log.Printf("[TemplateCache] Invalidated all cache entries")
}

// Stats returns current cache statistics for monitoring.
func (c *TemplateCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var expiredWorkflow, expiredStage, expiredActivity int

	for _, entry := range c.workflowTemplates {
		if entry.isExpired() {
			expiredWorkflow++
		}
	}
	for _, entry := range c.stageTemplates {
		if entry.isExpired() {
			expiredStage++
		}
	}
	for _, entry := range c.activityTemplates {
		if entry.isExpired() {
			expiredActivity++
		}
	}

	return CacheStats{
		WorkflowTemplates:        len(c.workflowTemplates),
		WorkflowTemplatesExpired: expiredWorkflow,
		StageTemplates:           len(c.stageTemplates),
		StageTemplatesExpired:    expiredStage,
		ActivityTemplates:        len(c.activityTemplates),
		ActivityTemplatesExpired: expiredActivity,
		TTL:                      c.ttl,
	}
}

// CacheStats holds cache statistics for monitoring.
type CacheStats struct {
	WorkflowTemplates        int
	WorkflowTemplatesExpired int
	StageTemplates           int
	StageTemplatesExpired    int
	ActivityTemplates        int
	ActivityTemplatesExpired int
	TTL                      time.Duration
}

// sortStageTemplatesByOrderIndex sorts stage templates by order_index in ascending order.
// Templates without order_index are placed at the end.
func sortStageTemplatesByOrderIndex(templates []*stagetemplatepb.StageTemplate) {
	n := len(templates)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			shouldSwap := false

			// Handle nil order_index (place at end)
			if templates[j].OrderIndex == nil && templates[j+1].OrderIndex != nil {
				shouldSwap = true
			} else if templates[j].OrderIndex != nil && templates[j+1].OrderIndex != nil {
				if *templates[j].OrderIndex > *templates[j+1].OrderIndex {
					shouldSwap = true
				}
			}

			if shouldSwap {
				templates[j], templates[j+1] = templates[j+1], templates[j]
			}
		}
	}
}
