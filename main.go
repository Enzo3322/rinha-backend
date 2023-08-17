package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
)

var db *gorm.DB
var err error

type Pessoa struct {
	ID         uint           `json:"id"`
	Apelido    string         `json:"apelido" gorm:"type:varchar(32);unique_index;not null"`
	Nome       string         `json:"nome" gorm:"type:varchar(100);not null"`
	Nascimento string         `json:"nascimento" gorm:"type:date;not null"`
	Stack      postgres.Jsonb `json:"stack" gorm:"type:jsonb"`
}

func main() {
	dsn := "postgresql://postgres:postgres@localhost:5432/rinha?sslmode=disable"
	db, err = gorm.Open("postgres", dsn)

	if err != nil {
		fmt.Println("Erro ao conectar ao banco de dados")
		log.Fatal(err)
	}
	defer db.Close()

	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	db.DB().SetConnMaxLifetime(time.Hour)

	db.AutoMigrate(&Pessoa{})

	r := gin.Default()

	r.POST("/pessoas", CreatePessoa)
	r.GET("/pessoas/:id", GetPessoa)
	r.GET("/pessoas", SearchPessoas)
	r.GET("/contagem-pessoas", CountPessoas)

	port := "9999"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	r.Run(":" + port)
}

func validatePessoa(pessoa *Pessoa) error {
	if pessoa.Apelido == "" || len(pessoa.Apelido) > 32 {
		return errors.New("Invalid apelido")
	}

	if pessoa.Nome == "" || len(pessoa.Nome) > 100 {
		return errors.New("Invalid nome")
	}

	_, err := time.Parse("2006-01-02", pessoa.Nascimento)
	if err != nil {
		return errors.New("Invalid nascimento")
	}

	return nil
}

func CreatePessoa(c *gin.Context) {
	var pessoa Pessoa
	if err := c.BindJSON(&pessoa); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Invalid data"})
		return
	}

	var existing Pessoa
	if err := db.Where("apelido = ?", pessoa.Apelido).First(&existing).Error; err == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Apelido already exists"})
		return
	}

	if err := validatePessoa(&pessoa); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	db.Create(&pessoa)
	c.JSON(http.StatusCreated, pessoa)
}

func GetPessoa(c *gin.Context) {
	var pessoa Pessoa
	id := c.Params.ByName("id")
	if err := db.Where("id = ?", id).First(&pessoa).Error; err != nil {
		c.AbortWithStatus(http.StatusNotFound)
	} else {
		c.JSON(http.StatusOK, pessoa)
	}
}

func SearchPessoas(c *gin.Context) {
	termo := c.DefaultQuery("t", "")
	var pessoas []Pessoa
	limit := 50

	query := db.Where("apelido ILIKE ?", "%"+termo+"%").
		Or("nome ILIKE ?", "%"+termo+"%").
		Or("stack::text ILIKE ?", "%"+termo+"%").
		Limit(limit).
		Find(&pessoas)

	if query.Error != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, pessoas)
}

func CountPessoas(c *gin.Context) {
	var count int
	if err := db.Model(&Pessoa{}).Count(&count).Error; err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	} else {
		c.JSON(http.StatusOK, gin.H{"count": count})
	}
}
