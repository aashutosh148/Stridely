package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/aashutosh148/Stridely/pacer-api/config"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/handlers"
	"github.com/aashutosh148/Stridely/pacer-api/jobs"
	"github.com/aashutosh148/Stridely/pacer-api/llm"
	"github.com/aashutosh148/Stridely/pacer-api/middleware"
	"github.com/aashutosh148/Stridely/pacer-api/services"
	"github.com/aashutosh148/Stridely/pacer-api/tools"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting pacer-api")

	// Load .env file in development
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	// Run database migrations
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	if cfg.SentryDSN != "" {
		slog.Info("SENTRY_DSN present but sentry is disabled; using structured logs only")
	}

	// Initialize database connection
	ctx := context.Background()
	postgres, err := db.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to connect to postgres:", err)
	}
	defer postgres.Close()

	// Initialize Redis connection (Session 6 - with Pool reference for RebuildWorkingMemory)
	var redis *db.Redis
	if cfg.RedisURL != "" {
		redis, err = db.NewRedis(ctx, cfg.RedisURL, postgres.Pool)
		if err != nil {
			slog.Warn("failed to connect to redis", "error", err)
		} else {
			defer redis.Close()
		}
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Pacer API",
		ErrorHandler: middleware.ErrorHandler,
	})

	// Global middleware
	app.Use(middleware.RecoveryMiddleware())
	app.Use(middleware.LoggingMiddleware())
	app.Use(middleware.CORSMiddleware(cfg.FrontendURL))

	// Initialize services
	stravaClient := services.NewStravaClient(postgres)
	analysisService := services.NewAnalysisService(postgres)
	planningService := services.NewPlanningService(postgres)
	raceService := services.NewRaceService()
	eventHub := services.NewEventHub()

	// Session 6 - Initialize memory service
	memoryService := services.NewMemoryService(postgres.Pool, cfg.OpenAIAPIKey)

	// Initialize LLM client based on provider configuration
	var llmClient llm.Client
	llmCfg := llm.Config{
		Provider: llm.Provider(cfg.LLMProvider),
		Model:    cfg.LLMModel,
	}

	// Set API key based on provider
	switch llm.Provider(cfg.LLMProvider) {
	case llm.ProviderAnthropic:
		llmCfg.APIKey = cfg.AnthropicAPIKey
		if llmCfg.APIKey == "" {
			slog.Warn("ANTHROPIC_API_KEY not set, agent features will be disabled")
		}
	case llm.ProviderOpenAI:
		llmCfg.APIKey = cfg.OpenAIAPIKey
		if llmCfg.APIKey == "" {
			slog.Warn("OPENAI_API_KEY not set, agent features will be disabled")
		}
	case llm.ProviderGemini:
		llmCfg.APIKey = cfg.GeminiAPIKey
		if llmCfg.APIKey == "" {
			slog.Warn("GEMINI_API_KEY not set, agent features will be disabled")
		}
	default:
		slog.Warn("unknown LLM_PROVIDER, defaulting to anthropic", "provider", cfg.LLMProvider)
		llmCfg.Provider = llm.ProviderAnthropic
		llmCfg.APIKey = cfg.AnthropicAPIKey
	}

	if llmCfg.APIKey != "" {
		var err error
		llmClient, err = llm.NewClient(llmCfg)
		if err != nil {
			slog.Warn("failed to initialize LLM client", "error", err, "provider", llmCfg.Provider)
		} else {
			slog.Info("llm client initialized", "provider", llmCfg.Provider, "model", llmClient.GetDefaultModel())
		}
	}

	// Initialize tool registry (Session 6 - pass memory service)
	toolRegistry := tools.NewRegistry(&tools.Dependencies{
		DB:       postgres,
		Redis:    redis,
		Strava:   stravaClient,
		Planning: planningService,
		Memory:   memoryService,
	})

	// Initialize agent service
	var agentService *services.AgentService
	if llmClient != nil && redis != nil {
		agentService = services.NewAgentService(
			postgres,
			redis,
			llmClient,
			toolRegistry,
			analysisService,
			memoryService,
		)
		slog.Info("agent service initialized")
	}

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(postgres)
	authHandler := handlers.NewAuthHandler(postgres, stravaClient, analysisService)
	webhookHandler := handlers.NewWebhookHandler(postgres, redis, stravaClient, analysisService, eventHub)
	activitiesHandler := handlers.NewActivitiesHandler(postgres, stravaClient, analysisService)
	fitnessHandler := handlers.NewFitnessHandler(analysisService)
	readinessHandler := handlers.NewReadinessHandler(postgres)
	statsHandler := handlers.NewStatsHandler(postgres)
	plansHandler := handlers.NewPlansHandler(postgres, planningService)
	raceHandler := handlers.NewRaceHandler(postgres, raceService)
	eventsHandler := handlers.NewEventsHandler(eventHub)

	var chatHandler *handlers.ChatHandler
	if agentService != nil {
		chatHandler = handlers.NewChatHandler(postgres, agentService)
	}

	// Health check route (no auth required)
	app.Get("/health", healthHandler.Check)

	// API routes (v1)
	api := app.Group("/api/v1")

	// Public routes (no auth)
	api.Get("/auth/strava", authHandler.StravaLogin)
	api.Get("/auth/strava/callback", authHandler.StravaCallback)

	// Webhook routes (no auth - Strava webhooks)
	api.Get("/webhooks/strava", webhookHandler.StravaVerify)
	api.Post("/webhooks/strava", webhookHandler.StravaWebhook)

	// Protected routes (auth required)
	protected := api.Use(middleware.AuthMiddleware())
	protected.Get("/auth/me", authHandler.Me)
	protected.Get("/activities", activitiesHandler.List)
	protected.Get("/activities/recent", activitiesHandler.Recent)
	protected.Get("/activities/:id", activitiesHandler.Get)
	protected.Post("/activities/sync", activitiesHandler.Sync)
	protected.Post("/activities/recalculate-zones", activitiesHandler.RecalculateZones)
	protected.Post("/activities/trigger-zone-recalc", activitiesHandler.TriggerZoneRecalc)
	protected.Get("/events/stream", eventsHandler.Stream)

	// Fitness metrics routes
	protected.Get("/fitness/metrics", fitnessHandler.GetMetrics)
	protected.Post("/fitness/threshold/estimate", fitnessHandler.EstimateThreshold)
	protected.Get("/readiness/today", readinessHandler.Today)
	protected.Get("/stats/overview", statsHandler.Overview)

	// Chat routes (Session 5)
	if chatHandler != nil {
		protected.Post("/chat", middleware.ChatRateLimit(100), chatHandler.Chat)
		protected.Get("/chat/history", chatHandler.GetHistory)
	}

	// Planning routes (Session 5)
	protected.Get("/plan/active", plansHandler.GetActive)
	protected.Get("/plan/week", plansHandler.GetWeek)
	protected.Post("/plan/generate", plansHandler.Generate)
	protected.Put("/plan/workout/:id", plansHandler.UpdateWorkout)
	protected.Post("/plan/adjust", plansHandler.Adjust)

	// Race routes
	protected.Post("/race/strategy", raceHandler.Strategy)
	protected.Get("/race/weather", raceHandler.Weather)
	protected.Post("/race/fueling", raceHandler.Fueling)

	// Background scheduler (Session 11)
	scheduler := jobs.SetupScheduler(&jobs.Dependencies{
		DB:       postgres,
		Redis:    redis,
		Planning: planningService,
		Agent:    agentService,
		Memory:   memoryService,
		Analysis: analysisService,
		Notifier: eventHub,
	})
	defer scheduler.Stop()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		slog.Info("shutting down server")
		if err := app.Shutdown(); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("server starting", "port", cfg.Port, "env", cfg.Env)
	if err := app.Listen(addr); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
