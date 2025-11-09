export interface WeatherReport {
    DbId: number,
    Time: Date,
    DeviceModel: string,
    TemperatureInF:  number,
    HumidityInPercentage: number
}
export interface HouseHvacRecommendation {
	DbId: number;
	Time: Date | string; // time.Time -> Date or ISO string
	ShouldOperateAirConditioner: boolean;
	TemperatureToSetAirConditionerInF: number;
	ShouldWindowBeOpen: boolean;
	WeatherDescription: string;
	IndoorTemperatureF: number;
	OutdoorTemperatureF: number;
}
