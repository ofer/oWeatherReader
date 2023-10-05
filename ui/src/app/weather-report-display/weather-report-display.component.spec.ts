import { ComponentFixture, TestBed } from '@angular/core/testing';

import { WeatherReportDisplayComponent } from './weather-report-display.component';

describe('WeatherReportDisplayComponent', () => {
  let component: WeatherReportDisplayComponent;
  let fixture: ComponentFixture<WeatherReportDisplayComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [WeatherReportDisplayComponent]
    });
    fixture = TestBed.createComponent(WeatherReportDisplayComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
