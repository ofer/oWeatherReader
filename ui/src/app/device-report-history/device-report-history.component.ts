import { Component, Input } from '@angular/core';
import { ApiService } from '../api.service';
import { EChartsOption } from 'echarts';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-device-report-history',
  templateUrl: './device-report-history.component.html',
  styleUrls: ['./device-report-history.component.scss']
})
export class DeviceReportHistoryComponent {
  // @Input() deviceModel!: string;

  private data: DataT[];
  private humidityData: DataH[];
  private DAYS_OF_HISTORY = 3;

  options: EChartsOption;
  updateOptions: EChartsOption;

  @Input()
  set deviceModel(value: string | null) {
    if (value != null) {
      // Call API to get data for the new device model
      this.api.getHistoricDataForDeviceModel(value).subscribe(historicWeatherReports => {
        // Update chart data with new data
        this.data = this.convertToTemperatureData(historicWeatherReports);
        this.humidityData = this.convertToHumidityData(historicWeatherReports);
        this.updateOptions = {
          series: [
            {
              data: this.data,
            },
            {
              data: this.humidityData
            }
          ],
        };
      });
    }
  }

  convertToTemperatureData(historicWeatherReports: WeatherReport[]): DataT[] {
    return historicWeatherReports.filter(report => this.isReportInRange(report)).map(report => {
      return {
        name: report.Time.toString(),
        value: [report.Time.toString(), report.TemperatureInF]
      } as DataT;
    });
  }

  convertToHumidityData(historicWeatherReports: WeatherReport[]): DataH[] {
    return historicWeatherReports.filter(report => this.isReportInRange(report)).map(report => {
      return {
        name: report.Time.toString(),
        value: [report.Time.toString(), report.HumidityInPercentage]
      } as DataH;
    });
  }

  isReportInRange(report: WeatherReport): boolean {
    const reportDate = new Date(report.Time);
    const oldestUseableDate = new Date();
    oldestUseableDate.setDate(oldestUseableDate.getDate() - this.DAYS_OF_HISTORY);
    return reportDate >= oldestUseableDate;
  }


  constructor(private api: ApiService) {
    this.data = [];
    this.humidityData = [];

    // initialize chart options:
    this.options = {
      title: {
        text: 'Weather Report History',
      },
      tooltip: {
        trigger: 'axis',
        // formatter: params => {
        //   params = params[0];
        //   const date = new Date(params.name);
        //   return (
        //     date.getDate() +
        //     '/' +
        //     (date.getMonth() + 1) +
        //     '/' +
        //     date.getFullYear() +
        //     ' : ' +
        //     params.value[1]
        //   );
        // },
        axisPointer: {
          animation: false,
        },
      },
      xAxis: {
        type: 'time',
        splitLine: {
          show: false,
        },
      },
      yAxis: {
        type: 'value',
        boundaryGap: [0, '100%'],
        splitLine: {
          show: false,
        },
      },
      series: [
        {
          name: 'Temperature Data',
          type: 'line',
          showSymbol: false,
          data: this.data,
        },
        {
          name: 'Humidity Data',
          type: 'line',
          showSymbol: false,
          data: this.humidityData,
        }
      ],
    };
    this.updateOptions = {
      series: [
        {
          data: this.data,
        },
        {
          data: this.humidityData
        }
      ],
    };

  }

  ngOnInit(): void {
    // // Mock dynamic data:
    // this.timer = setInterval(() => {
    //   for (let i = 0; i < 5; i++) {
    //     this.data.shift();
    //     this.data.push(this.randomData());
    //   }

    //   // update series data:
    // }, 1000);
  }
}

type DataT = {
  name: string;
  value: [string, number];
};

type DataH = {
  name: string;
  value: [string, number];
};
