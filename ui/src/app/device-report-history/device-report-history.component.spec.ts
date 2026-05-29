import { ComponentFixture, TestBed, waitForAsync, fakeAsync, tick } from '@angular/core/testing';
import { DeviceReportHistoryComponent } from './device-report-history.component';
import { ApiService } from '../api.service';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { WeatherReport } from '../weather-report';
import { WeatherReportEvent } from '../weather-report-event';
import { EChartsOption } from 'echarts';
import { of, Subject } from 'rxjs';
import { NgxEchartsModule } from 'ngx-echarts';

class MockApiService {
  historicDataSubject = new Subject<WeatherReport[]>();
  eventsSubject = new Subject<WeatherReportEvent>();

  getHistoricDataForDeviceModel(model: string) {
    return this.historicDataSubject.asObservable();
  }

  get weatherReportEvents$() {
    return this.eventsSubject.asObservable();
  }
}

describe('DeviceReportHistoryComponent', () => {
  let component: DeviceReportHistoryComponent;
  let fixture: ComponentFixture<DeviceReportHistoryComponent>;
  let mockApi: MockApiService;

  const mockReports: WeatherReport[] = [
    { DbId: 1, Time: new Date(Date.now() - 86400000), DeviceModel: 'Bresser-3CH', TemperatureInF: 72, HumidityInPercentage: 45 },
    { DbId: 2, Time: new Date(Date.now() - 43200000), DeviceModel: 'Bresser-3CH', TemperatureInF: 74, HumidityInPercentage: 48 },
    { DbId: 3, Time: new Date(Date.now() - 21600000), DeviceModel: 'Bresser-3CH', TemperatureInF: 76, HumidityInPercentage: 50 },
  ];

  beforeEach(waitForAsync(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule, NgxEchartsModule.forRoot({ echarts: () => import('echarts') })],
      declarations: [DeviceReportHistoryComponent],
      providers: [{ provide: ApiService, useClass: MockApiService }]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(DeviceReportHistoryComponent);
    component = fixture.componentInstance;
    mockApi = TestBed.inject(ApiService) as unknown as MockApiService;
    fixture.detectChanges();
  });

  afterEach(() => {
    mockApi.historicDataSubject.complete();
    mockApi.eventsSubject.complete();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('loads history when deviceModel is assigned', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    expect(component.data.length).toBe(mockReports.length);
    expect(component.humidityData.length).toBe(mockReports.length);
  }));

  it('updates updateOptions.series[0].data and series[1].data from API history', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    const series = component.updateOptions.series as unknown as any[];
    const tempData = series?.[0]?.data;
    const humData = series?.[1]?.data;

    expect(tempData).toEqual(component.data);
    expect(humData).toEqual(component.humidityData);
    expect(tempData.length).toBe(mockReports.length);
    expect(humData.length).toBe(mockReports.length);
  }));

  it('refreshes history when a matching server event arrives', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    const updatedReports: WeatherReport[] = [
      { DbId: 1, Time: new Date(Date.now() - 86400000), DeviceModel: 'Bresser-3CH', TemperatureInF: 72, HumidityInPercentage: 45 },
      { DbId: 2, Time: new Date(Date.now() - 43200000), DeviceModel: 'Bresser-3CH', TemperatureInF: 74, HumidityInPercentage: 48 },
      { DbId: 3, Time: new Date(Date.now() - 21600000), DeviceModel: 'Bresser-3CH', TemperatureInF: 76, HumidityInPercentage: 50 },
      { DbId: 4, Time: new Date(Date.now() - 10800000), DeviceModel: 'Bresser-3CH', TemperatureInF: 78, HumidityInPercentage: 52 },
    ];

    const event: WeatherReportEvent = { dbId: 4, time: 1704117600, deviceModel: 'Bresser-3CH' };
    mockApi.eventsSubject.next(event);
    tick();

    mockApi.historicDataSubject.next(updatedReports);
    tick();

    expect(component.data.length).toBe(updatedReports.length);
    const series = component.updateOptions.series as unknown as any[];
    expect(series[0].data.length).toBe(updatedReports.length);
  }));

  it('does not refresh history when a non-matching server event arrives', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    const initialTempCount = component.data.length;

    const event: WeatherReportEvent = { dbId: 99, time: 1704117600, deviceModel: 'OtherModel' };
    mockApi.eventsSubject.next(event);
    tick();

    // Should NOT trigger a data update from the events subject
    expect(component.data.length).toBe(initialTempCount);
  }));

  it('switches cleanly when deviceModel changes', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    expect(component.currentModel).toBe('Bresser-3CH');
    expect(component.data.length).toBeGreaterThan(0);
  }));

  it('unsubscribes from event streams on destroy', fakeAsync(() => {
    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(mockReports);
    tick();

    const initialTempCount = component.data.length;

    component.ngOnDestroy();

    const event: WeatherReportEvent = { dbId: 99, time: 1704117600, deviceModel: 'Bresser-3CH' };
    mockApi.eventsSubject.next(event);
    tick();

    expect(component.data.length).toBe(initialTempCount);
  }));

  it('filters reports outside the configured history window', fakeAsync(() => {
    const oldReports: WeatherReport[] = [
      { DbId: 1, Time: new Date('2020-01-01T10:00:00'), DeviceModel: 'Bresser-3CH', TemperatureInF: 70, HumidityInPercentage: 40 },
    ];

    component.deviceModel = 'Bresser-3CH';
    tick();

    mockApi.historicDataSubject.next(oldReports);
    tick();

    expect(component.data.length).toBe(0);
    expect(component.humidityData.length).toBe(0);
  }));
});
