import { Component } from '@angular/core';
import { LatestWeatherReporterService } from '../latest-weather-reporter.service';
import { timer } from 'rxjs';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-home-summary-report',
  templateUrl: './home-summary-report-component.component.html',
  styleUrls: ['./home-summary-report-component.component.scss']
})
export class HomeSummaryReportComponentComponent {

  weatherReports: WeatherReport[] = [];

  constructor(private latestWeatherReporter: LatestWeatherReporterService) { 
    timer(0, 10000).subscribe(() => {
      console.log('Latest reports:', this.latestWeatherReporter.latestWeatherReports || 'No reports');
      this.weatherReports = this.latestWeatherReporter.latestWeatherReports;
    });
  }
}
