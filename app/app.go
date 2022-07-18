package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/antoniodipinto/ikisocket"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/fatih/color"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	middlewares "github.com/nikola43/kasiopea/middleware"
	websockets "github.com/nikola43/kasiopea/websockets"
	"github.com/nikola43/web3golanghelper/web3helper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var httpServer *fiber.App

type App struct {
	web3GolangHelper *web3helper.Web3GolangHelper
}

func (a *App) Initialize(port string) {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	e := InitializeHttpServer().Listen(port)
	if e != nil {
		log.Fatal(e)
	}
}

func (a *App) InitWeb3() {
	pk := "b366406bc0b4883b9b4b3b41117d6c62839174b7d21ec32a5ad0cc76cb3496bd"
	rpcUrl := "https://speedy-nodes-nyc.moralis.io/84a2745d907034e6d388f8d6/avalanche/testnet"
	wsUrl := "wss://speedy-nodes-nyc.moralis.io/84a2745d907034e6d388f8d6/avalanche/testnet/ws"
	a.web3GolangHelper = web3helper.NewWeb3GolangHelper(rpcUrl, wsUrl, pk)

	chainID, err := a.web3GolangHelper.HttpClient().NetworkID(context.Background())
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Chain Id: " + chainID.String())

	proccessEvents(a.web3GolangHelper)
}

func proccessEvents(web3GolangHelper *web3helper.Web3GolangHelper) {
	nodeAddress := "0x2Fcd73952e53aAd026c378F378812E5bb069eF6E"
	nodeAbi, _ := abi.JSON(strings.NewReader(string(NodeManagerV83.NodeManagerV83ABI)))
	fmt.Println(color.YellowString("  ----------------- Blockchain Events -----------------"))
	fmt.Println(color.CyanString("\tListen node manager address: "), color.GreenString(nodeAddress))
	logs := make(chan types.Log)
	sub := BuildContractEventSubscription(web3GolangHelper, nodeAddress, logs)

	for {
		select {
		case err := <-sub.Err():
			fmt.Println(err)
			//out <- err.Error()

		case vLog := <-logs:
			fmt.Println("paco")
			fmt.Println("vLog.TxHash: " + vLog.TxHash.Hex())
			res, err := nodeAbi.Unpack("GiftCardPayed", vLog.Data)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(res)
			services.SetGiftCardIntentPayment(res[2].(string))

		}
	}
}

func BuildContractEventSubscription(web3GolangHelper *web3helper.Web3GolangHelper, contractAddress string, logs chan types.Log) ethereum.Subscription {

	query := ethereum.FilterQuery{
		Addresses: []common.Address{common.HexToAddress(contractAddress)},
	}

	sub, err := web3GolangHelper.WebSocketClient().SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		fmt.Println(sub)
	}
	return sub
}

func (a *App) ListenBridgeEvents() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	app.Listen(":3020")
}

func InitializeHttpServer() *fiber.App {
	httpServer = fiber.New(fiber.Config{
		BodyLimit: 2000 * 1024 * 1024, // this is the default limit of 4MB
	})

	httpServer.Use(middlewares.XApiKeyMiddleware)

	httpServer.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	httpServer.Get("/", controllers.ON)

	ws := httpServer.Group("/ws")

	// Setup the middleware to retrieve the data sent in first GET request
	ws.Use(middlewares.WebSocketUpgradeMiddleware)

	// Pull out in another function
	// all the ikisocket callbacks and listeners
	setupSocketListeners()

	ws.Get("/:id", ikisocket.New(func(kws *ikisocket.Websocket) {
		websockets.SocketInstance = kws

		// Retrieve the user id from endpoint
		userId := kws.Params("id")

		// Add the connection to the list of the connected clients
		// The UUID is generated randomly and is the key that allow
		// ikisocket to manage Emit/EmitTo/Broadcast
		models.SocketClients[userId] = kws.UUID

		// Every websocket connection has an optional session key => value storage
		kws.SetAttribute("user_id", userId)

		//Broadcast to all the connected users the newcomer
		// kws.Broadcast([]byte(fmt.Sprintf("New user connected: %s and UUID: %s", userId, kws.UUID)), true)
		//Write welcome message
		kws.Emit([]byte(fmt.Sprintf("Socket connected")))
	}))

	fmt.Println(color.YellowString("  ----------------- Websockets -----------------"))
	fmt.Println(color.CyanString("\t    Websocket URL: "), color.GreenString("ws://127.0.0.1:3000/ws"))

	api := httpServer.Group("/api") // /api
	v1 := api.Group("/v1")          // /api/v1
	HandleRoutes(v1)

	/*
		err := httpServer.Listen(port)
		if err != nil {
			log.Fatal(err)
		}
	*/

	return httpServer
}

func InitializeDatabase(user, password, database_name string) {
	connectionString := fmt.Sprintf(
		"%s:%s@/%s?parseTime=true",
		user,
		password,
		database_name,
	)

	DB, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	database.GormDB, err = gorm.Open(mysql.New(mysql.Config{Conn: DB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatal(err)
	}
}

// Setup all the ikisocket listeners
// pulled out main function
func setupSocketListeners() {

	// Multiple event handling supported
	ikisocket.On(ikisocket.EventConnect, func(ep *ikisocket.EventPayload) {
		fmt.Println(color.GreenString("  New On Connection "), color.CyanString("User: "), color.YellowString(ep.Kws.GetStringAttribute("user_id")))
		fmt.Println("")
	})

	// On message event
	ikisocket.On(ikisocket.EventMessage, func(ep *ikisocket.EventPayload) {
		socketUserId := ep.Kws.GetStringAttribute("user_id")
		fmt.Println(color.YellowString("  New On Message Event "), color.CyanString("User: "), color.YellowString(socketUserId))
		fmt.Println(string(ep.Data))
		fmt.Println("")
	})

	// On disconnect event
	ikisocket.On(ikisocket.EventDisconnect, func(ep *ikisocket.EventPayload) {
		fmt.Println(color.RedString("  New On Disconnect Event "), color.CyanString("User: "), color.YellowString(ep.Kws.GetStringAttribute("user_id")))
		fmt.Println("")
		delete(models.SocketClients, ep.Kws.GetStringAttribute("user_id"))
	})

	// On close event
	// This event is called when the server disconnects the user actively with .Close() method
	ikisocket.On(ikisocket.EventClose, func(ep *ikisocket.EventPayload) {
		fmt.Println(color.RedString("  New On Close Event "), color.CyanString("User: "), color.YellowString(ep.Kws.GetStringAttribute("user_id")))
		fmt.Println("")

		delete(models.SocketClients, ep.Kws.GetStringAttribute("user_id"))
	})

	// On error event
	ikisocket.On(ikisocket.EventError, func(ep *ikisocket.EventPayload) {
		fmt.Println(color.RedString("  New On Error Event "), color.CyanString("User: "), color.YellowString(ep.Kws.GetStringAttribute("user_id")))
		fmt.Println(color.CyanString("\tUser: "), color.YellowString(ep.Kws.GetStringAttribute("user_id")))
		fmt.Println("")
	})
}
