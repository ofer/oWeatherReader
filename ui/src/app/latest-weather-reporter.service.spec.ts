import { TestBed } from '@angular/core/testing';

import { LatestWeatherReporterService } from './latest-weather-reporter.service';

describe('LatestWeatherReporterService', () => {
  let service: LatestWeatherReporterService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(LatestWeatherReporterService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
