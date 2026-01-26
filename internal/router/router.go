package router

import (
	"chat-backend/internal/config"
	"chat-backend/internal/database"
	"chat-backend/internal/handlers"
	"chat-backend/internal/middleware"
	"chat-backend/internal/utils"
	"chat-backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, db *database.Database, hub *websocket.Hub, cfg *config.Config) {
	// Health check endpoint
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Initialize Twilio service
	twilioService := utils.NewTwilioService(cfg)

	api := r.Group("/api/v1")
	
	// Auth routes
	auth := api.Group("/auth")
	{
		authHandler := handlers.NewAuthHandler(db, twilioService)
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/qr/:user_id", authHandler.GetQRCode)
		auth.POST("/verify-phone", authHandler.VerifyPhone)
		auth.POST("/send-code", authHandler.SendCode)
		auth.POST("/verify-code", authHandler.VerifyCode)
		auth.POST("/register-with-code", authHandler.RegisterWithCode)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// User routes
		userHandler := handlers.NewUserHandler(db)
		user := protected.Group("/users")
		{
			user.GET("/me", userHandler.GetMe)
			user.PUT("/me", userHandler.UpdateMe)
			user.PUT("/location", userHandler.UpdateLocation)
			user.GET("/nearby", userHandler.GetNearbyUsers)
			user.GET("/search", userHandler.SearchByUsername)
			user.GET("/devices", userHandler.GetDevices)
		}
		
		// Public user search (no auth required for username search)
		public := api.Group("/public")
		{
			public.GET("/users/search", userHandler.SearchByUsername)
		}

		// Contact routes
		contactHandler := handlers.NewContactHandler(db)
		contacts := protected.Group("/contacts")
		{
			contacts.GET("", contactHandler.GetContacts)
			contacts.POST("/scan", contactHandler.ScanQRCode)
			contacts.DELETE("/:contact_id", contactHandler.DeleteContact)
		}

		// Chat routes
		chatHandler := handlers.NewChatHandler(db, hub)
		chats := protected.Group("/chats")
		{
			chats.GET("", chatHandler.GetChats)
			chats.POST("", chatHandler.CreateChat)
			chats.GET("/:chat_id", chatHandler.GetChat)
			chats.GET("/:chat_id/messages", chatHandler.GetMessages)
			chats.POST("/:chat_id/messages", chatHandler.SendMessage)
		}

		// Message routes
		messageHandler := handlers.NewMessageHandler(db, hub)
		messages := protected.Group("/messages")
		{
			messages.PUT("/:message_id", messageHandler.EditMessage)
			messages.DELETE("/:message_id", messageHandler.DeleteMessage)
			messages.POST("/:message_id/forward", messageHandler.ForwardMessage)
			messages.POST("/:message_id/reaction", messageHandler.AddReaction)
			messages.DELETE("/:message_id/reaction", messageHandler.RemoveReaction)
			messages.POST("/read", messageHandler.MarkAsRead)
			messages.POST("/:message_id/pin", messageHandler.PinMessage)
			messages.DELETE("/:message_id/pin", messageHandler.UnpinMessage)
			messages.POST("/:message_id/poll/vote", messageHandler.VotePoll)
			messages.GET("/search", messageHandler.SearchMessages)
			messages.GET("/:message_id/translate", messageHandler.TranslateMessage)
		}

		// Typing indicator routes
		typingHandler := handlers.NewTypingHandler(db, hub)
		typing := protected.Group("/typing")
		{
			typing.POST("/:chat_id", typingHandler.SetTyping)
			typing.GET("/:chat_id", typingHandler.GetTyping)
		}

		// Group routes
		groupHandler := handlers.NewGroupHandler(db)
		groups := protected.Group("/groups")
		{
			groups.POST("", groupHandler.CreateGroup)
			groups.GET("", groupHandler.GetGroups)
			groups.GET("/:group_id", groupHandler.GetGroup)
			groups.PUT("/:group_id", groupHandler.UpdateGroup)
			groups.DELETE("/:group_id", groupHandler.DeleteGroup)
			groups.POST("/:group_id/members", groupHandler.AddMember)
			groups.DELETE("/:group_id/members/:member_id", groupHandler.RemoveMember)
			groups.GET("/:group_id/statistics", groupHandler.GetStatistics)
		}

		// Channel routes
		channelHandler := handlers.NewChannelHandler(db)
		channels := protected.Group("/channels")
		{
			channels.POST("", channelHandler.CreateChannel)
			channels.POST("/:channel_id/subscribe", channelHandler.Subscribe)
			channels.POST("/:channel_id/unsubscribe", channelHandler.Unsubscribe)
			channels.POST("/:channel_id/messages/:message_id/view", channelHandler.RecordView)
			channels.GET("/:channel_id/statistics", channelHandler.GetStatistics)
		}

		// Proposal routes
		proposalHandler := handlers.NewProposalHandler(db)
		proposals := protected.Group("/proposals")
		{
			proposals.POST("", proposalHandler.CreateProposal)
			proposals.GET("", proposalHandler.GetProposals)
			proposals.PUT("/:proposal_id/accept", proposalHandler.AcceptProposal)
			proposals.PUT("/:proposal_id/reject", proposalHandler.RejectProposal)
		}

		// Call routes
		callHandler := handlers.NewCallHandler(db, hub)
		calls := protected.Group("/calls")
		{
			calls.POST("", callHandler.InitiateCall)
			calls.POST("/:call_id/answer", callHandler.AnswerCall)
			calls.POST("/:call_id/end", callHandler.EndCall)
		}

		// File upload routes
		fileHandler := handlers.NewFileHandler(db)
		files := protected.Group("/files")
		{
			files.POST("/upload", fileHandler.UploadFile)
			files.GET("/:filename", fileHandler.ServeFile)
		}

		// Settings routes
		settingsHandler := handlers.NewSettingsHandler(db)
		settings := protected.Group("/settings")
		{
			settings.GET("", settingsHandler.GetSettings)
			settings.PUT("", settingsHandler.UpdateSettings)
			settings.PUT("/account", settingsHandler.UpdateAccountSettings)
			settings.PUT("/privacy", settingsHandler.UpdatePrivacySettings)
			settings.PUT("/chat", settingsHandler.UpdateChatSettings)
			settings.PUT("/notifications", settingsHandler.UpdateNotificationSettings)
			settings.PUT("/appearance", settingsHandler.UpdateAppearanceSettings)
			settings.PUT("/data", settingsHandler.UpdateDataSettings)
			settings.PUT("/calls", settingsHandler.UpdateCallSettings)
			settings.PUT("/groups", settingsHandler.UpdateGroupSettings)
			settings.PUT("/advanced", settingsHandler.UpdateAdvancedSettings)
			settings.GET("/sessions", settingsHandler.GetSessions)
			settings.DELETE("/sessions/:session_id", settingsHandler.TerminateSession)
			settings.POST("/block", settingsHandler.BlockUser)
			settings.DELETE("/block/:user_id", settingsHandler.UnblockUser)
			settings.GET("/blocked", settingsHandler.GetBlockedUsers)
			settings.POST("/suspend", settingsHandler.SuspendAccount)
			settings.POST("/delete", settingsHandler.DeleteAccount)
			settings.POST("/cache/clear", settingsHandler.ClearCache)
			settings.GET("/data-usage", settingsHandler.GetDataUsage)
		}

		// Product routes
		productHandler := handlers.NewProductHandler(db)
		products := protected.Group("/products")
		{
			products.POST("", productHandler.CreateProduct)
			products.GET("", productHandler.GetProducts)
			products.GET("/:product_id", productHandler.GetProduct)
			products.PUT("/:product_id", productHandler.UpdateProduct)
			products.DELETE("/:product_id", productHandler.DeleteProduct)
			products.GET("/user/:user_id", productHandler.GetUserProducts)
		}

		// Comment routes
		commentHandler := handlers.NewCommentHandler(db)
		comments := protected.Group("/products/:product_id/comments")
		{
			comments.POST("", commentHandler.CreateComment)
			comments.GET("", commentHandler.GetComments)
		}
		protected.DELETE("/comments/:comment_id", commentHandler.DeleteComment)
		protected.POST("/comments/:comment_id/report", commentHandler.ReportSpam)

		// Like routes
		likeHandler := handlers.NewLikeHandler(db)
		protected.POST("/products/:product_id/like", likeHandler.LikeProduct)
		protected.DELETE("/products/:product_id/like", likeHandler.UnlikeProduct)
		protected.POST("/comments/:comment_id/like", likeHandler.LikeComment)
		protected.DELETE("/comments/:comment_id/like", likeHandler.UnlikeComment)
		protected.GET("/products/:product_id/likes", likeHandler.GetProductLikes)
	}

	// WebSocket route
	r.GET("/ws", func(c *gin.Context) {
		websocket.HandleWebSocket(hub, c, db)
	})
}

