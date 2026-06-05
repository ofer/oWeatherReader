import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ApiService, EVENT_SOURCE_TOKEN } from './api.service';
import { WeatherReportEvent } from './weather-report-event';
import { WeatherReport } from './weather-report';
import { DeviceModel } from './device-model';

class MockEventSource {
  static instances: MockEventSource[] = [];
  url!: string;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  closeCalls = 0;

  constructor(url: string) {
    MockEventSource.instances.push(this);
    this.url = url;
  }

  close(): void {
    this.closeCalls++;
  }

  emitMessage(data: string): void {
    if (this.onmessage) {
      this.onmessage({ data });
    }
  }

  emitError(): void {
    if (this.onerror) {
      this.onerror(new Event('error'));
    }
  }

  static reset(): void {
    MockEventSource.instances = [];
  }
}

describe('ApiService', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    MockEventSource.reset();

    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [
        ApiService,
        { provide: EVENT_SOURCE_TOKEN, useValue: MockEventSource }
      ]
    });

    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('creates no EventSource before subscription', () => {
    expect(MockEventSource.instances.length).toBe(0);
  });

  it('opens EventSource on subscription', () => {
    service.weatherReportEvents$.subscribe();
    expect(MockEventSource.instances.length).toBe(1);
    expect(MockEventSource.instances[0].url).toBe('./reports/events');
  });

  it('emits parsed weather report events', () => {
    const event: WeatherReportEvent = { dbId: 42, time: 1684516800, deviceModel: 'Bresser-3CH' };
    let emitted: WeatherReportEvent | undefined;

    service.weatherReportEvents$.subscribe(e => { emitted = e; });
    MockEventSource.instances[0].emitMessage(JSON.stringify(event));

    expect(emitted).toEqual(event);
  });

  it('reports invalid JSON as an error', () => {
    let errorReceived: unknown;

    service.weatherReportEvents$.subscribe({
      error: err => { errorReceived = err; }
    });
    MockEventSource.instances[0].emitMessage('not-json');

    expect(errorReceived).toBeDefined();
  });

  it('closes EventSource on unsubscribe', (done) => {
    const subscription = service.weatherReportEvents$.subscribe();
    expect(MockEventSource.instances[0].closeCalls).toBe(0);

    subscription.unsubscribe();

    expect(MockEventSource.instances[0].closeCalls).toBe(1);
    done();
  });

  it('shares one connection across multiple subscribers', () => {
    const sub1 = service.weatherReportEvents$.subscribe();
    const sub2 = service.weatherReportEvents$.subscribe();

    expect(MockEventSource.instances.length).toBe(1);

    const event: WeatherReportEvent = { dbId: 1, time: 0, deviceModel: 'test' };
    MockEventSource.instances[0].emitMessage(JSON.stringify(event));

    sub1.unsubscribe();
    sub2.unsubscribe();
  });

  it('existing REST methods call the same endpoints as before', () => {
    const model = 'Bresser-3CH';
    const reports: WeatherReport[] = [
      {
        DbId: 1,
        Time: new Date('2024-01-01'),
        DeviceModel: model,
        TemperatureInF: 72,
        HumidityInPercentage: 45
      }
    ];

    service.getHistoricDataForDeviceModel(model).subscribe(data => {
      expect(data).toEqual(reports);
    });

    const req = httpMock.expectOne(`./reports/${model}`);
    expect(req.request.method).toBe('GET');
    req.flush(reports);
  });

  it('getModels calls the models endpoint', () => {
    const models: DeviceModel[] = [
      { DeviceModel: 'Bresser-3CH', ReportCount: 100, Name: 'Bresser 3 Channel Weather Station', isIndoor: false, isOutdoor: false }
    ];

    service.getModels().subscribe(data => {
      expect(data).toEqual(models);
    });

    const req = httpMock.expectOne('./models');
    expect(req.request.method).toBe('GET');
    req.flush(models);
  });

  it('getLatestRecommendedReport calls the correct endpoint', () => {
    const recommendation = {
      DbId: 1,
      Time: new Date(),
      ShouldOperateAirConditioner: true,
      TemperatureToSetAirConditionerInF: 72,
      ShouldWindowBeOpen: false,
      WeatherDescription: 'Sunny',
      IndoorTemperatureF: 75,
      OutdoorTemperatureF: 85
    };

    service.getLatestRecommendedReport().subscribe(data => {
      expect(data).toEqual(recommendation);
    });

    const req = httpMock.expectOne('./recommendations/latest');
    expect(req.request.method).toBe('GET');
    req.flush(recommendation);
  });

  it('EventSource emits to all active subscribers', (done) => {
    let receivedCount = 0;

    const event: WeatherReportEvent = { dbId: 1, time: 0, deviceModel: 'test' };
    const sub1 = service.weatherReportEvents$.subscribe(e => { receivedCount++; });
    const sub2 = service.weatherReportEvents$.subscribe(e => { receivedCount++; });

    MockEventSource.instances[0].emitMessage(JSON.stringify(event));

    setTimeout(() => {
      expect(receivedCount).toBe(2);
      sub1.unsubscribe();
      sub2.unsubscribe();
      done();
    }, 10);
  });
});
