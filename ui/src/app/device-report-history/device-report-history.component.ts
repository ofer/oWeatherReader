import { Component, Input, OnDestroy, OnInit } from '@angular/core';
import { ApiService } from '../api.service';
import { EChartsOption } from 'echarts';
import { WeatherReport } from '../weather-report';
import { switchMap, takeUntil, Subscription, Subject } from 'rxjs';

@Component({
  selector: 'app-device-report-history',
  templateUrl: './device-report-history.component.html',
  styleUrls: ['./device-report-history.component.scss']
})
export class DeviceReportHistoryComponent implements OnInit, OnDestroy {

  data: DataT[];
  humidityData: DataH[];
  private DAYS_OF_HISTORY = 3;

  options: EChartsOption;
  updateOptions: EChartsOption;

  currentModel: string | null = null;
  private subscriptions = new Subscription();
  private destroy$ = new Subject<void>();

  @Input()
  set deviceModel(value: string | null) {
    if (value != null) {
      this.currentModel = value;
      this.loadHistory(value);
    }
  }

  private loadHistory(model: string): void {
    this.api.getHistoricDataForDeviceModel(model).subscribe(historicWeatherReports => {
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
        scale: true,
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
    const eventSubscription = this.api.weatherReportEvents$
      .pipe(
        switchMap(event => {
          if (event.deviceModel === this.currentModel) {
            return this.api.getHistoricDataForDeviceModel(this.currentModel!);
          }
          return [];
        }),
        takeUntil(this.destroy$)
      )
      .subscribe(historicWeatherReports => {
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

    this.subscriptions.add(eventSubscription);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.subscriptions.unsubscribe();
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
