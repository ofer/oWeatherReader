import { Component } from '@angular/core';
import { LatestWeatherReporterService } from '../latest-weather-reporter.service';
import { timer } from 'rxjs';
import { WeatherReport, HouseHvacRecommendation } from '../weather-report';
import { ApiService } from '../api.service';

@Component({
  selector: 'app-home-summary-report',
  templateUrl: './home-summary-report-component.component.html',
  styleUrls: ['./home-summary-report-component.component.scss']
})
export class HomeSummaryReportComponentComponent {

  weatherReports: WeatherReport[] = [];
  recommendations: HouseHvacRecommendation | null = null;

  constructor(private latestWeatherReporter: LatestWeatherReporterService, private apiService: ApiService) { 
    timer(0, 10000).subscribe(() => {
      console.log('Latest reports:', this.latestWeatherReporter.latestWeatherReports || 'No reports');
      this.weatherReports = this.latestWeatherReporter.latestWeatherReports;
      this.apiService.getLatestRecommendedReport().subscribe(rec => {
        this.recommendations = rec;
      });
    });
  }
}
