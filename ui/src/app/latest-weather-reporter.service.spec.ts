import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule } from '@angular/common/http/testing';

import { LatestWeatherReporterService } from './latest-weather-reporter.service';

describe('LatestWeatherReporterService', () => {
  let service: LatestWeatherReporterService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule]
    });
    service = TestBed.inject(LatestWeatherReporterService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
