package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)




func main() {
	e := echo.New()
	e.GET("/data", GetData)
	e.GET("/init", Initialize)
	e.Logger.Fatal(e.Start(":8000"))
}




type Response struct {
	CasesInState  string `json:"cases_in_state" `
	CasesInCountry string `json:"cases_in_country" `
	LastUpdated string `json:"last_updated" `
}




func GetData(c echo.Context) error {
	latitude := c.QueryParam("Lat")
	longitude := c.QueryParam("Long")

	clientOptions := options.Client().
		ApplyURI("mongodb+srv://admin:admin@cluster0.4dels.mongodb.net/Data?retryWrites=true&w=majority")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	database := client.Database("Data")
	StateData := database.Collection("stateData")

	url := "https://trueway-geocoding.p.rapidapi.com/ReverseGeocode?location="+latitude+"%2C" + longitude + "&language=en"

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("x-rapidapi-host", "trueway-geocoding.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", "6b6b1d8691msh2ccdbf5f1f13699p1054dfjsn464b3ee9832d")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	address := result["results"].([]interface{})
	addressJson := address[0].(map[string]interface{})
	state := addressJson["region"]
	var stateTuple bson.M
	if err = StateData.FindOne(ctx, bson.M{"State" : state}).Decode(&stateTuple); err != nil {
		return err
	}
	var totalTuple bson.M
	if err = StateData.FindOne(ctx, bson.M{"State" : "Total"}).Decode(&totalTuple); err != nil {
		return err
	}
	cases := stateTuple["Cases"]
	totalCases := totalTuple["Cases"]
	timestamp := totalTuple["Timestamp"]
	r := &Response{
		CasesInState: fmt.Sprint(cases),
		CasesInCountry: fmt.Sprint(totalCases),
		LastUpdated: fmt.Sprint(timestamp),
	}
	return c.JSON(http.StatusOK, r)
}




func Initialize(c echo.Context) error  {

	clientOptions := options.Client().
		ApplyURI("mongodb+srv://admin:admin@cluster0.4dels.mongodb.net/Data?retryWrites=true&w=majority")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	database := client.Database("Data")
	State_data := database.Collection("stateData")
	url := "https://data.covid19india.org/csv/latest/state_wise.csv"
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	lines, _ := csv.NewReader(resp.Body).ReadAll()
	if err = State_data.Drop(ctx); err != nil {
		return err
	}
	for _,line := range lines {
		_, err := State_data.InsertOne(ctx , bson.D{
			{Key: "State", Value: line[0]},
			{Key: "Cases", Value: line[1]},
			{Key: "Timestamp", Value: time.Now()},
		})
		if err != nil {
			return err
		}
	}
	return c.String(http.StatusOK, fmt.Sprintln("Initialized"))
}
