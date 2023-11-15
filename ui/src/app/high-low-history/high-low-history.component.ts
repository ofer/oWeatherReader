import { Component, Input } from '@angular/core';
import { ApiService } from '../api.service';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-high-low-history',
  templateUrl: './high-low-history.component.html',
  styleUrls: ['./high-low-history.component.scss']
})
export class HighLowHistoryComponent {

  highAndLowTableData: HighAndLowData[];
  highAndLowTableColumns: string[] = ['date', 'highTemperatureF', 'lowTemperatureF', 'highHumidity', 'lowHumidity'];

  constructor(private api: ApiService) {
    this.highAndLowTableData = [];
  }

  @Input()
  set deviceModel(value: string | null) {
    if (value != null) {
      // Call API to get data for the new device model
      this.api.getHistoricDataForDeviceModel(value).subscribe(historicWeatherReports => {
        // Update chart data with new data
        this.highAndLowTableData = this.getHighAndLowTableData(historicWeatherReports);
      });
    }
  }

  getHighAndLowTableData(historicWeatherReports: WeatherReport[]): HighAndLowData[] {
    const highAndLowTableData: HighAndLowData[] = [];
    const groupedByDate = this.groupByDate(historicWeatherReports);
    const groupedByMiddleOfTheDay = this.groupByMiddleOfTheDay(historicWeatherReports);
    groupedByDate.forEach((reports, date) => {
      const highTemperatureF = Math.max(...reports.map(report => report.TemperatureInF));
      const lowTemperatureF = Math.min(...(groupedByMiddleOfTheDay.get(date) ?? []).map(report => report.TemperatureInF));
      const highHumidity = Math.max(...(groupedByMiddleOfTheDay.get(date) ?? []).map(report => report.HumidityInPercentage));
      const lowHumidity = Math.min(...reports.map(report => report.HumidityInPercentage));
      highAndLowTableData.push({
        date: new Date(date),
        highTemperatureF: highTemperatureF,
        lowTemperatureF: lowTemperatureF,
        highHumidity: highHumidity,
        lowHumidity: lowHumidity
      });
    });
    return highAndLowTableData;
  }

  groupByDate(historicWeatherReports: WeatherReport[]) {
    const groupedByDate = new Map<string, WeatherReport[]>();
    historicWeatherReports.forEach(report => {
      const reportDate = new Date(report.Time);
      const date = new Date(reportDate.getFullYear(), reportDate.getMonth(), reportDate.getDate());
      const dateString = date.toDateString();
      if (!groupedByDate.has(dateString)) {
        groupedByDate.set(dateString, []);
      }
      groupedByDate.get(dateString)?.push(report);
    });
    return groupedByDate;
  }

  groupByMiddleOfTheDay(historicWeatherReports: WeatherReport[]) {
    const groupedByDate = new Map<string, WeatherReport[]>();
    historicWeatherReports.forEach(report => {
      let reportDate = new Date(report.Time);
      reportDate = new Date(reportDate.getTime() - 12 * 60 * 60000);
      const date = new Date(reportDate.getFullYear(), reportDate.getMonth(), reportDate.getDate());
      const dateString = date.toDateString();
      if (!groupedByDate.has(dateString)) {
        groupedByDate.set(dateString, []);
      }
      groupedByDate.get(dateString)?.push(report);
    });
    return groupedByDate;
  }
}

type HighAndLowData = {
  date: Date;
  highTemperatureF: number;
  lowTemperatureF: number;
  highHumidity: number;
  lowHumidity: number;
};