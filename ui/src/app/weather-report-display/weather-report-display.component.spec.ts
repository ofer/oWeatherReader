import { ComponentFixture, TestBed } from '@angular/core/testing';

import { WeatherReportDisplayComponent } from './weather-report-display.component';
import { WeatherReport } from '../weather-report';

describe('WeatherReportDisplayComponent', () => {
  let component: WeatherReportDisplayComponent;
  let fixture: ComponentFixture<WeatherReportDisplayComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [WeatherReportDisplayComponent]
    });
    fixture = TestBed.createComponent(WeatherReportDisplayComponent);
    component = fixture.componentInstance;
    component.report = {
      DbId: 1,
      Time: new Date('2024-01-01T00:00:00Z'),
      DeviceModel: 'TestModel',
      TemperatureInF: 72,
      HumidityInPercentage: 45,
    } as WeatherReport;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
