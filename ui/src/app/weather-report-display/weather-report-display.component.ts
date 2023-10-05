import { Component, Input } from '@angular/core';
import { WeatherReport } from '../weather-report';

@Component({
  selector: 'app-weather-report-display',
  templateUrl: './weather-report-display.component.html',
  styleUrls: ['./weather-report-display.component.scss']
})
export class WeatherReportDisplayComponent {
  @Input()
  report!: WeatherReport;
}
