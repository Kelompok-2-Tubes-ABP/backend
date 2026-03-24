package handler

import (
	"time"

	"financeapi/essentials/models"
	"financeapi/essentials/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SavingsGoalHandler struct {
	service *services.SavingsGoalService
}

func NewSavingsGoalHandler(service *services.SavingsGoalService) *SavingsGoalHandler {
	return &SavingsGoalHandler{service: service}
}

func (h *SavingsGoalHandler) CreateSavingsGoal() gin.HandlerFunc {
	return func(c *gin.Context) {
		var goal models.SavingsGoal
		if err := c.ShouldBindJSON(&goal); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		goal.UserID, _ = primitive.ObjectIDFromHex(userID.(string))

		createdGoal, err := h.service.CreateSavingsGoal(goal)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, createdGoal)
	}
}

func (h *SavingsGoalHandler) GetSavingsGoal() gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid goal ID"})
			return
		}

		userID, _ := c.Get("user_id")
		goal, err := h.service.GetSavingsGoal(goalID, userID.(string))
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"goal":           goal,
			"progress":       goal.CalculateProgress(),
			"monthly_needed": goal.CalculateMonthlyNeeded(),
			"on_track":       goal.IsOnTrack(),
		})
	}
}

func (h *SavingsGoalHandler) GetUserSavingsGoals() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		goals, err := h.service.GetUserSavingsGoals(userID.(string))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		goalsWithProgress := make([]gin.H, len(goals))
		for i, goal := range goals {
			goalsWithProgress[i] = gin.H{
				"goal":           goal,
				"progress":       goal.CalculateProgress(),
				"monthly_needed": goal.CalculateMonthlyNeeded(),
				"on_track":       goal.IsOnTrack(),
			}
		}

		c.JSON(200, goalsWithProgress)
	}
}

func (h *SavingsGoalHandler) UpdateSavingsGoal() gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid goal ID"})
			return
		}

		var updates map[string]interface{}
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		goal, err := h.service.UpdateSavingsGoal(goalID, userID.(string), updates)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, goal)
	}
}

func (h *SavingsGoalHandler) AddContribution() gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid goal ID"})
			return
		}

		var req struct {
			Amount float64 `json:"amount" binding:"required"`
			Note   string  `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request format"})
			return
		}

		userID, _ := c.Get("user_id")
		userIDObj, _ := primitive.ObjectIDFromHex(userID.(string))
		contribution := models.SavingsContribution{
			SavingsGoalID: goalID,
			UserID:        userIDObj,
			Amount:        req.Amount,
			Note:          req.Note,
		}

		result, err := h.service.AddContribution(contribution)
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, result)
	}
}

func (h *SavingsGoalHandler) GetContributions() gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid goal ID"})
			return
		}

		userID, _ := c.Get("user_id")
		contributions, err := h.service.GetGoalContributions(goalID, userID.(string))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, contributions)
	}
}

func (h *SavingsGoalHandler) DeleteSavingsGoal() gin.HandlerFunc {
	return func(c *gin.Context) {
		goalID, err := primitive.ObjectIDFromHex(c.Param("id"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid goal ID"})
			return
		}

		userID, _ := c.Get("user_id")
		err = h.service.DeleteSavingsGoal(goalID, userID.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "Savings goal deleted successfully"})
	}
}

func (h *SavingsGoalHandler) GetSavingsSummary() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		summary, err := h.service.GetSavingsSummary(userID.(string))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, summary)
	}
}

func CreateGoalWithDate(name, description string, targetAmount float64, targetDate string, category string) models.SavingsGoal {
	t, _ := time.Parse("2006-01-02", targetDate)
	return models.SavingsGoal{
		Name:         name,
		Description:  description,
		TargetAmount: targetAmount,
		TargetDate:   t,
		Category:     category,
		Priority:     2,
	}
}
